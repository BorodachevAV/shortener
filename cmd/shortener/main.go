package main

import (
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

type ShortenJsonRequest struct {
	URL string
}

type ShortenJsonRResponse struct {
	result string
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
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

	// делаем регистратор SugaredLogger
	var sugar = *logger.Sugar()

	logFn := func(w http.ResponseWriter, r *http.Request) {
		// функция Now() возвращает текущее время
		start := time.Now()

		// эндпоинт /ping
		uri := r.RequestURI
		// метод запроса
		method := r.Method

		// точка, где выполняется хендлер pingHandler
		h.ServeHTTP(w, r) // обслуживание оригинального запроса

		// Since возвращает разницу во времени между start
		// и моментом вызова Since. Таким образом можно посчитать
		// время выполнения запроса.
		duration := time.Since(start)

		// отправляем сведения о запросе в zap
		sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
		)

	}
	// возвращаем функционально расширенный хендлер
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

	//читаем тело
	urlFromRequest, _ := io.ReadAll(r.Body)
	reader := bytes.NewReader(urlFromRequest)
	gzreader, _ := gzip.NewReader(reader)
	output, _ := io.ReadAll(gzreader)
	//генерим короткий url
	shortURL := randString(8)
	// сохраняем в мапе
	_, err := url.Parse(string(output))
	if err != nil {
		http.Error(w, "not url", http.StatusBadRequest)
		return
	}
	urlsStorage.Store(shortURL, string(output))
	//заполняем ответ
	body := fmt.Sprintf("%s/%s", cfg.Cfg.BaseURL, shortURL)
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", cfg.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(body))
	if err != nil {
		log.Println(err.Error())
	}
}

func shortenJson(w http.ResponseWriter, r *http.Request) {
	var req ShortenJsonRequest
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
	flag.Parse()
	if cfg.Cfg.ServerAddress == "" {
		cfg.Cfg.ServerAddress = *a
	}
	if cfg.Cfg.BaseURL == "" {
		cfg.Cfg.BaseURL = *b
	}
	r := chi.NewRouter()
	r.Use(withLogging)
	r.Use(gzipHandle)
	r.Post(`/`, shorten)
	r.Post(`/api/shorten`, shortenJson)
	r.Get(`/{id}`, expand)
	log.Fatal(http.ListenAndServe(cfg.Cfg.ServerAddress, r))
}
