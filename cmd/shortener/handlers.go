package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BorodachevAV/shortener/internal/auth"
	"github.com/BorodachevAV/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type ShortenerHandler struct {
	storage storage.ShortenerStorage
}

type ShortenJSONRequest struct {
	URL string
}

type ShortenBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ShortenBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func WriteData(ss storage.ShortenerStorage, sd *storage.ShortenerData) error {
	return ss.WriteURL(sd)
}

func ReadData(ss storage.ShortenerStorage, s string) (*storage.ShortenerData, error) {
	return ss.ReadURL(s)
}

func WriteBatchData(ss storage.ShortenerStorage, sd []*storage.ShortenerData) error {
	return ss.WriteBatch(sd)
}

func GetUserURLs(ss storage.ShortenerStorage, userID string) ([]*storage.ShortenerData, error) {
	return ss.GetUserURLs(userID)
}

func (sh ShortenerHandler) shorten(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", conf.Cfg.ServerAddress)
	var userIDValue string
	userID, err := r.Cookie("UserID")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			userID, err := auth.BuildJWTString()
			if err != nil {
				log.Println(err)
			}
			cookies := &http.Cookie{
				Name:  "UserID",
				Value: userID,
			}
			http.SetCookie(w, cookies)
			userIDValue = userID
		} else {
			log.Println(err)
		}
	} else {
		userIDValue = strings.TrimSpace(userID.Value)
	}

	output, _ := io.ReadAll(r.Body)
	//читаем тело
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader := bytes.NewReader(output)
		gzreader, err := gzip.NewReader(reader)
		if err != nil {
			log.Println(err.Error())
		}
		output, err = io.ReadAll(gzreader)
		if err != nil {
			log.Println(err.Error())
		}
	}

	strOutput := string(output)
	//генерим короткий url
	shortURL := randString(8)
	// сохраняем в мапе
	_, err = url.Parse(strOutput)
	if err != nil {
		http.Error(w, "not url", http.StatusBadRequest)
		return
	}
	sd := &storage.ShortenerData{
		UserID:      auth.GetUserId(userIDValue),
		ShortURL:    fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL),
		OriginalURL: string(output),
	}
	err = WriteData(sh.storage, sd)
	if errors.Is(err, storage.ErrDuplicate) {
		w.WriteHeader(http.StatusConflict)
		body, _ := sh.storage.CheckDuplicateURL(sd.OriginalURL)
		_, err = w.Write([]byte(body))
		if err != nil {
			log.Println(err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	body := sd.ShortURL
	_, err = w.Write([]byte(body))
	if err != nil {
		log.Println(err.Error())
	}
}

func (sh ShortenerHandler) shortenBatch(w http.ResponseWriter, r *http.Request) {
	var batch []ShortenBatchRequest
	var buf bytes.Buffer
	var sdBatch []*storage.ShortenerData
	var Response []ShortenBatchResponse

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.Unmarshal(buf.Bytes(), &batch)
	for _, URL := range batch {
		shortURL := randString(8)
		sdBatch = append(sdBatch, &storage.ShortenerData{
			ShortURL:    fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL),
			OriginalURL: URL.OriginalURL,
		})
		Response = append(Response, ShortenBatchResponse{
			CorrelationID: URL.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL),
		})
	}
	WriteBatchData(sh.storage, sdBatch)

	respBody, _ := json.Marshal(Response)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Host", conf.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(respBody)
	if err != nil {
		log.Println(err.Error())
	}

}

func (sh ShortenerHandler) shortenJSON(w http.ResponseWriter, r *http.Request) {
	var req ShortenJSONRequest
	var buf bytes.Buffer

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Host", conf.Cfg.ServerAddress)

	var userIDValue string
	userID, err := r.Cookie("UserID")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			userID, err := auth.BuildJWTString()
			if err != nil {
				log.Println(err)
			}
			cookies := &http.Cookie{
				Name:  "UserID",
				Value: userID,
			}
			http.SetCookie(w, cookies)
			userIDValue = userID
		} else {
			log.Println(err)
		}
	} else {
		userIDValue = userID.Value
	}

	// читаем тело запроса

	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//генерим короткий url
	shortURL := randString(8)
	// сохраняем в мапе

	_, err = url.Parse(req.URL)
	if err != nil {
		http.Error(w, "not url", http.StatusBadRequest)
		return
	}

	sd := &storage.ShortenerData{
		UserID:      auth.GetUserId(userIDValue),
		ShortURL:    fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL),
		OriginalURL: req.URL,
	}
	err = WriteData(sh.storage, sd)
	if errors.Is(err, storage.ErrDuplicate) {
		w.WriteHeader(http.StatusConflict)
		sd.ShortURL, err = sh.storage.CheckDuplicateURL(sd.OriginalURL)
		if err != nil {
			log.Println(err.Error())
		}
	}
	w.WriteHeader(http.StatusCreated)
	//заполняем ответ
	body := sd.ShortURL
	m := make(map[string]string)
	m["result"] = body
	respBody, _ := json.Marshal(m)
	_, err = w.Write(respBody)
	if err != nil {
		log.Println(err.Error())
	}
}

func (sh ShortenerHandler) expand(w http.ResponseWriter, r *http.Request) {
	shortURL := fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, chi.URLParam(r, "id"))
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	val, err := ReadData(sh.storage, shortURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if val != nil {
		w.Header().Add("Location", val.OriginalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		http.Error(w, "short url not found", http.StatusNotFound)
		return
	}
}

func (sh ShortenerHandler) getUserURLs(w http.ResponseWriter, r *http.Request) {
	var Response []UserURLResponse
	userID, err := r.Cookie("UserID")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	val, err := GetUserURLs(sh.storage, auth.GetUserId(userID.Value))
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
	}
	for _, v := range val {
		Response = append(Response, UserURLResponse{
			v.ShortURL,
			v.OriginalURL,
		})
	}
	respBody, _ := json.Marshal(Response)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Host", conf.Cfg.ServerAddress)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(respBody)
	if err != nil {
		log.Println(err.Error())
	}
}

func pingDB(w http.ResponseWriter, r *http.Request) {
	pg, _ := pgConnect(conf.Cfg.DataBaseDNS)
	err := pg.Ping()
	if err != nil {
		http.Error(w, "wrong conn string", http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}

}
