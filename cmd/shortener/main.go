package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/BorodachevAV/shortener/internal/config"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// алфавит для коротких url
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// генерим сид
var (
	urlsStorage sync.Map
	seededRand  *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg                    = config.New()
)

type ShortenJSONRequest struct {
	URL string
}

type ShortenerData struct {
	ID          uint   `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type StorageWriter struct {
	file *os.File
}

func NewStorageWriter(filename string) (*StorageWriter, error) {
	// открываем файл для записи в конец
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &StorageWriter{file: file}, nil
}

func (w *StorageWriter) WriteData(event *ShortenerData) error {
	data, err := json.Marshal(&event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = w.file.Write(data)
	return err
}

func (w *StorageWriter) Close() error {
	// закрываем файл
	return w.file.Close()
}

type StorageReader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewStorageReader(filename string) (*StorageReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &StorageReader{file: file,
		scanner: bufio.NewScanner(file),
	}, nil
}

func (r *StorageReader) ReadData() (*ShortenerData, error) {
	if !r.scanner.Scan() {
		return nil, r.scanner.Err()
	}
	data := r.scanner.Bytes()

	storageData := ShortenerData{}
	err := json.Unmarshal(data, &storageData)
	if err != nil {
		return nil, err
	}

	return &storageData, nil
}

func (r *StorageReader) Close() error {
	// закрываем файл
	return r.file.Close()
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Add("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func withLogging(h http.Handler) http.Handler {
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer logger.Sync()

	var sugar = *logger.Sugar()

	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		uri := r.RequestURI

		method := r.Method

		h.ServeHTTP(w, r)

		duration := time.Since(start)

		sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
		)

	}
	return http.HandlerFunc(logFn)
}

// генерим короткий url
func randString(length int) string {
	randB := make([]byte, length)
	for i := range randB {
		randB[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(randB)
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
	urlsStorage.Store(shortURL, string(output))
	//заполняем ответ
	if cfg.Cfg.FileStoragePath != "" {
		file := cfg.Cfg.FileStoragePath
		storageReader, _ := NewStorageReader(file)
		data, _ := storageReader.ReadData()
		if data != nil {
			var newData ShortenerData
			newData.ID = data.ID + 1
			newData.ShortURL = shortURL
			newData.OriginalURL = string(output)
			storageWriter, _ := NewStorageWriter(file)
			err = storageWriter.WriteData(&newData)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			var newData ShortenerData
			newData.ID = 1
			newData.ShortURL = shortURL
			newData.OriginalURL = string(output)
			storageWriter, _ := NewStorageWriter(file)
			err = storageWriter.WriteData(&newData)
			if err != nil {
				log.Println(err.Error())
			}
		}

	}

	body := fmt.Sprintf("%s/%s", cfg.Cfg.BaseURL, shortURL)
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", cfg.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(body))
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
	urlsStorage.Store(shortURL, req.URL)
	if cfg.Cfg.FileStoragePath != "" {
		file := cfg.Cfg.FileStoragePath
		storageReader, _ := NewStorageReader(file)
		data, _ := storageReader.ReadData()
		if data != nil {
			var newData ShortenerData
			newData.ID = data.ID + 1
			newData.ShortURL = shortURL
			newData.OriginalURL = req.URL
			storageWriter, _ := NewStorageWriter(file)
			err = storageWriter.WriteData(&newData)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			var newData ShortenerData
			newData.ID = 1
			newData.ShortURL = shortURL
			newData.OriginalURL = req.URL
			storageWriter, _ := NewStorageWriter(file)
			err = storageWriter.WriteData(&newData)
			if err != nil {
				log.Println(err.Error())
			}
		}

	}
	//заполняем ответ
	body := fmt.Sprintf("%s/%s", cfg.Cfg.BaseURL, shortURL)
	m := make(map[string]string)
	m["result"] = body
	respBody, _ := json.Marshal(m)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Host", cfg.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(respBody)
	if err != nil {
		log.Println(err.Error())
	}
}
func expand(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	val, ok := urlsStorage.Load(shortURL)

	if ok {
		w.Header().Add("Location", val.(string))
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		http.Error(w, "short url not found", http.StatusNotFound)
		return
	}
}
func main() {

	a := flag.String("a", "localhost:8080", "shortener host")
	b := flag.String("b", "http://localhost:8080", "response host")
	f := flag.String("f", "", "storage file path")
	flag.Parse()
	if cfg.Cfg.ServerAddress == "" {
		cfg.Cfg.ServerAddress = *a
	}
	if cfg.Cfg.BaseURL == "" {
		cfg.Cfg.BaseURL = *b
	}
	if cfg.Cfg.FileStoragePath == "" {
		cfg.Cfg.FileStoragePath = *f
	}
	r := chi.NewRouter()
	r.Use(withLogging)
	r.Use(gzipHandle)
	r.Post(`/`, shorten)
	r.Post(`/api/shorten`, shortenJSON)
	r.Get(`/{id}`, expand)
	log.Fatal(http.ListenAndServe(cfg.Cfg.ServerAddress, r))
}
