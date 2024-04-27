package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// алфавит для коротких url
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// генерим сид
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// генерим короткий url
func randString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(b)
}

// создаем хранилище url
var urlsStorage = make(map[string]string)

// хендлер запросов
func shortener(w http.ResponseWriter, r *http.Request) {
	//проверка метода, обрабатываем только GET и POST
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Only POST or GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	//обработка POST
	if r.Method == http.MethodPost {
		//читаем тело
		urlFromRequest, _ := io.ReadAll(r.Body)
		//генерим короткий url
		shortUrl := randString(8)
		// сохраняем в мапе
		urlsStorage[shortUrl] = string(urlFromRequest)
		//заполняем ответ
		body := fmt.Sprintf("http://localhost:8080/%s", shortUrl)
		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Host", "localhost:8080")
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(body))
		if err != nil {
			panic(err)
		}
	}
	//обработка GET
	if r.Method == http.MethodGet {
		//делим url на части
		shortUrl := strings.Split(r.RequestURI, "/")
		//если адрес не сходится с ожидаемым форматом отдаем 400
		if len(shortUrl) != 2 || shortUrl[1] == "" {
			http.Error(w, "Bad Request, not short url", http.StatusBadRequest)
			return
		}
		//проверяем в мапе наличие ключа, отдаем 404 если его нет
		val, ok := urlsStorage[shortUrl[1]]
		if ok {
			w.Header().Add("Location", val)
		} else {
			http.Error(w, "short url not found", http.StatusNotFound)
			return
		}
		//редиректим на полный адрес из мапы
		w.WriteHeader(http.StatusTemporaryRedirect)
	}

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, shortener)
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
