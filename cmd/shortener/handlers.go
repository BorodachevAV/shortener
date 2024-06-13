package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
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
	Correlation_id string `json:"correlation_id"`
	Original_url   string `json:"original_url"`
}

type ShortenBatchResponse struct {
	Correlation_id string `json:"correlation_id"`
	Short_url      string `json:"short_url"`
}

func WriteData(ss ShortenerStorage, sd *ShortenerData) error {
	return ss.WriteURL(sd)
}

func ReadData(ss ShortenerStorage, s string) (*ShortenerData, bool) {
	return ss.ReadURL(s)
}

func WriteBatchData(ss ShortenerStorage, sd []*ShortenerData) error {
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
	sd := ShortenerData{
		ShortURL:    shortURL,
		OriginalURL: string(output),
	}
	if conf.Cfg.DataBaseDNS != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		db, err := NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
		if err != nil {
			log.Println(err.Error())
		}
		WriteData(db, &sd)
	}
	WriteData(mapStorage, &sd)

	//заполняем ответ
	log.Println(conf.Cfg.FileStoragePath)
	if conf.Cfg.FileStoragePath != "" {
		file := conf.Cfg.FileStoragePath
		fileStorage, _ := NewFileStorage(file)
		data, _ := ReadData(fileStorage, shortURL)

		if data != nil {
			var newData ShortenerData
			newData.ID = data.ID + 1
			newData.ShortURL = shortURL
			newData.OriginalURL = string(output)

			err = WriteData(fileStorage, &newData)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			var newData ShortenerData
			newData.ID = 1
			newData.ShortURL = shortURL
			newData.OriginalURL = string(output)
			err = WriteData(fileStorage, &newData)
			if err != nil {
				log.Println(err.Error())
			}
		}

	}

	body := fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL)
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
	var sdBatch []*ShortenerData
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
	db, err := NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
	if err != nil {
		log.Println(err.Error())
	}
	for _, URL := range batch {
		shortURL := randString(8)
		sdBatch = append(sdBatch, &ShortenerData{
			ShortURL:    shortURL,
			OriginalURL: URL.Original_url,
		})
		Response = append(Response, ShortenBatchResponse{
			Correlation_id: URL.Correlation_id,
			Short_url:      shortURL,
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

	json.Unmarshal(buf.Bytes(), &req)

	//генерим короткий url
	shortURL := randString(8)
	// сохраняем в мапе

	_, err = url.Parse(req.URL)
	if err != nil {
		http.Error(w, "not url", http.StatusBadRequest)
		return
	}

	sd := ShortenerData{
		ShortURL:    shortURL,
		OriginalURL: req.URL,
	}
	if conf.Cfg.DataBaseDNS != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		db, err := NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
		if err != nil {
			log.Println(err.Error())
		}
		WriteData(db, &sd)
	}
	WriteData(mapStorage, &sd)
	if conf.Cfg.FileStoragePath != "" {
		file := conf.Cfg.FileStoragePath
		fileStorage, _ := NewFileStorage(file)
		data, _ := ReadData(fileStorage, shortURL)
		if data != nil {
			var newData ShortenerData
			newData.ID = data.ID + 1
			newData.ShortURL = shortURL
			newData.OriginalURL = req.URL
			err = WriteData(fileStorage, &newData)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			var newData ShortenerData
			newData.ID = 1
			newData.ShortURL = shortURL
			newData.OriginalURL = req.URL
			err = WriteData(fileStorage, &newData)
			if err != nil {
				log.Println(err.Error())
			}
		}

	}
	//заполняем ответ
	body := fmt.Sprintf("%s/%s", conf.Cfg.BaseURL, shortURL)
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
	shortURL := chi.URLParam(r, "id")
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	if conf.Cfg.DataBaseDNS != "" {
		ctx := context.Background()
		db, err := NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
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
