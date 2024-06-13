package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BorodachevAV/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ShortenJSONRequest struct {
	URL string
}

type ShortenBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortenBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func WriteData(ss storage.ShortenerStorage, sd *storage.ShortenerData) error {
	return ss.WriteURL(sd)
}

func ReadData(ss storage.ShortenerStorage, s string) (*storage.ShortenerData, bool) {
	return ss.ReadURL(s)
}

func WriteBatchData(ss storage.ShortenerStorage, sd []*storage.ShortenerData) error {
	for _, data := range sd {
		err := ss.WriteURL(data)
		if err != nil {
			return err
		}
	}
	return nil
}

func shorten(w http.ResponseWriter, r *http.Request) {
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
	_, err := url.Parse(strOutput)
	if err != nil {
		http.Error(w, "not url", http.StatusBadRequest)
		return
	}
	log.Println(string(output))
	sd := &storage.ShortenerData{
		ShortURL:    fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL),
		OriginalURL: string(output),
	}
	if conf.Cfg.DataBaseDNS != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		db, err := storage.NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
		if err != nil {
			log.Println(err.Error())
		}
		err = WriteData(db, sd)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				sd, _ = db.GetShortURLByOriginal(sd.OriginalURL)
			} else {
				log.Println(err.Error())
			}
		}
	} else if conf.Cfg.FileStoragePath != "" {
		file := conf.Cfg.FileStoragePath
		fileStorage, _ := storage.NewFileStorage(file)
		data, _ := ReadData(fileStorage, shortURL)

		if data != nil {
			var newData storage.ShortenerData
			newData.ID = data.ID + 1
			newData.ShortURL = shortURL
			newData.OriginalURL = string(output)

			err = WriteData(fileStorage, &newData)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			var newData storage.ShortenerData
			newData.ID = 1
			newData.ShortURL = shortURL
			newData.OriginalURL = string(output)
			err = WriteData(fileStorage, &newData)
			if err != nil {
				log.Println(err.Error())
			}
		}
		WriteData(mapStorage, sd)
	} else {
		WriteData(mapStorage, sd)
	}

	body := sd.ShortURL
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", conf.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(body))
	if err != nil {
		log.Println(err.Error())
	}
}

func shortenBatch(w http.ResponseWriter, r *http.Request) {
	var batch []ShortenBatchRequest
	var buf bytes.Buffer
	var sdBatch []*storage.ShortenerData
	var Response []ShortenBatchResponse
	// читаем тело запроса

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.Unmarshal(buf.Bytes(), &batch)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	db, err := storage.NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
	if err != nil {
		log.Println(err.Error())
	}
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
	WriteBatchData(db, sdBatch)

	respBody, _ := json.Marshal(Response)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Host", conf.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(respBody)
	if err != nil {
		log.Println(err.Error())
	}

}

func shortenJSON(w http.ResponseWriter, r *http.Request) {
	var req ShortenJSONRequest
	var buf bytes.Buffer
	// читаем тело запроса

	_, err := buf.ReadFrom(r.Body)
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
		ShortURL:    fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL),
		OriginalURL: req.URL,
	}
	if conf.Cfg.DataBaseDNS != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		db, err := storage.NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
		if err != nil {
			log.Println(err.Error())
		}
		err = WriteData(db, sd)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				sd, _ = db.GetShortURLByOriginal(sd.OriginalURL)
			} else {
				log.Println(err.Error())
			}
		}
	} else if conf.Cfg.FileStoragePath != "" {
		file := conf.Cfg.FileStoragePath
		fileStorage, _ := storage.NewFileStorage(file)
		data, _ := ReadData(fileStorage, shortURL)
		if data != nil {
			sd.ID = data.ID + 1
			err = WriteData(fileStorage, sd)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			sd.ID = 1
			err = WriteData(fileStorage, sd)
			if err != nil {
				log.Println(err.Error())
			}
		}
		WriteData(mapStorage, sd)
	} else {
		WriteData(mapStorage, sd)
	}

	//заполняем ответ
	body := sd.ShortURL
	m := make(map[string]string)
	m["result"] = body
	respBody, _ := json.Marshal(m)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Host", conf.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(respBody)
	if err != nil {
		log.Println(err.Error())
	}
}

func expand(w http.ResponseWriter, r *http.Request) {
	shortURL := fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, chi.URLParam(r, "id"))
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	if conf.Cfg.DataBaseDNS != "" {
		ctx := context.Background()
		db, err := storage.NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
		if err != nil {
			log.Println(err.Error())
		}
		val, ok := ReadData(db, shortURL)
		if ok {
			w.Header().Add("Location", val.OriginalURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "short url not found", http.StatusNotFound)
			return
		}
	} else {
		val, ok := ReadData(mapStorage, shortURL)

		if ok {
			w.Header().Add("Location", val.OriginalURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "short url not found", http.StatusNotFound)
			return
		}
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
