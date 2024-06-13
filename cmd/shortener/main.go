package main

import (
	"compress/gzip"
	"context"
	"flag"
	"github.com/BorodachevAV/shortener/internal/config"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// алфавит для коротких url
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// генерим сид
var (
	mapStorage = MapStorage{urlsStorage: &sync.Map{}}
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	conf       = config.NewConfig()
)

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

func main() {

	a := flag.String("a", "localhost:8080", "shortener host")
	b := flag.String("b", "http://localhost:8080", "response host")
	f := flag.String("f", "", "storage file path")
	d := flag.String("d", "", "db connect string")

	flag.Parse()

	if conf.Cfg.ServerAddress == "" {
		conf.Cfg.ServerAddress = *a
	}
	if conf.Cfg.BaseURL == "" {
		conf.Cfg.BaseURL = *b
	}
	if conf.Cfg.FileStoragePath == "" {
		conf.Cfg.FileStoragePath = *f
	}
	if conf.Cfg.DataBaseDNS == "" {
		conf.Cfg.DataBaseDNS = *d
	}
	if conf.Cfg.DataBaseDNS != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		db, err := NewDBStorage(conf.Cfg.DataBaseDNS, ctx)
		if err != nil {
			log.Println(err.Error())
		}
		db.createSchema()
	}
	r := chi.NewRouter()
	r.Use(withLogging)
	r.Use(gzipHandle)
	r.Post(`/`, shorten)
	r.Post(`/api/shorten`, shortenJSON)
	r.Post(`/api/shorten/batch`, shortenBatch)
	r.Get(`/{id}`, expand)
	r.Get(`/ping`, pingDB)

	log.Fatal(http.ListenAndServe(conf.Cfg.ServerAddress, r))
}
