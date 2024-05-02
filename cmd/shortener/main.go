package main

import (
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Config struct {
	serverAddress string `env:"SERVER_ADDRESS"`
	baseUrl       string `env:"BASE_URL"`
}

// алфавит для коротких url
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// генерим сид
var (
	urlsStorage            = make(map[string]string)
	seededRand  *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg         Config
	a           = flag.String("a", "localhost:8080", "shortener host")
	b           = flag.String("b", "localhost:8080", "response host")
)

// генерим короткий url
func randString(length int) string {
	randB := make([]byte, length)
	for i := range randB {
		randB[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(randB)
}

func shorten(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	//читаем тело
	urlFromRequest, _ := io.ReadAll(r.Body)
	//генерим короткий url
	shortUrl := randString(8)
	// сохраняем в мапе
	urlsStorage[shortUrl] = string(urlFromRequest)
	//заполняем ответ
	if cfg.serverAddress == "" {
		cfg.serverAddress = *a
	}
	if cfg.baseUrl == "" {
		cfg.baseUrl = *b
	}
	body := fmt.Sprintf("http://%s/%s", cfg.baseUrl, shortUrl)
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", fmt.Sprintf("%s", cfg.serverAddress))
	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte(body))
	if err != nil {
		panic(err)
	}
}
func expand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	shortUrl := chi.URLParam(r, "id")
	fmt.Println(fmt.Sprintf("id is %s", shortUrl))
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	val, ok := urlsStorage[shortUrl]

	if ok {
		w.Header().Add("Location", val)
	} else {
		http.Error(w, "short url not found", http.StatusNotFound)
		w.Header().Add("Location", fmt.Sprint(urlsStorage))
		return
	}
	//редиректим на полный адрес из мапы
	w.WriteHeader(http.StatusTemporaryRedirect)
}
func main() {
	flag.Parse()
	if cfg.serverAddress == "" {
		cfg.serverAddress = *a
	}
	r := chi.NewRouter()
	r.Post(`/`, shorten)
	r.Get(`/{id}`, expand)
	log.Fatal(http.ListenAndServe(cfg.serverAddress, r))

}
