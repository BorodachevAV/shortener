package storage

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"os"
	"sync"
)

type ShortenerData struct {
	ID          uint   `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ShortenerStorage interface {
	WriteURL(*ShortenerData) error
	ReadURL(shortURL string) (*ShortenerData, bool)
}

type MapStorage struct {
	UrlsStorage *sync.Map
}

func (f MapStorage) WriteURL(data *ShortenerData) error {
	f.UrlsStorage.Store(data.ShortURL, data.OriginalURL)
	return nil
}

func (f MapStorage) ReadURL(URL string) (*ShortenerData, bool) {
	val, ok := f.UrlsStorage.Load(URL)
	if !ok {
		return nil, false
	}
	resp := ShortenerData{
		OriginalURL: val.(string),
	}
	return &resp, true
}

type FileStorage struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewFileStorage(filename string) (*FileStorage, error) {
	// открываем файл для записи в конец
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &FileStorage{file: file,
		scanner: bufio.NewScanner(file)}, nil
}

func (f FileStorage) WriteURL(sd *ShortenerData) error {
	data, err := json.Marshal(&sd)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = f.file.Write(data)
	return err
}

func (f FileStorage) ReadURL(shortURL string) (*ShortenerData, bool) {
	if !f.scanner.Scan() {
		return nil, false
	}
	data := f.scanner.Bytes()

	storageData := ShortenerData{}
	err := json.Unmarshal(data, &storageData)
	if err != nil {
		return nil, false
	}

	return &storageData, false
}

type DBStorage struct {
	db  *sql.DB
	ctx context.Context
}

func NewDBStorage(DNS string, ctx context.Context) (*DBStorage, error) {
	db, err := sql.Open("pgx", DNS)
	if err != nil {
		log.Println(err.Error())
	}
	return &DBStorage{
		db:  db,
		ctx: ctx,
	}, nil
}

func (db DBStorage) CreateSchema() error {

	tx, err := db.db.BeginTx(db.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start a transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("failed to rollback the transaction: %v", err)
			}
		}
	}()

	createShema :=
		`CREATE TABLE IF NOT EXISTS url_storage(
			short_url VARCHAR(200) PRIMARY KEY,
			original_url VARCHAR(200) NOT NULL UNIQUE
		)`

	if _, err := tx.ExecContext(db.ctx, createShema); err != nil {
		return fmt.Errorf("failed to execute statement `%s`: %w", createShema, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the transaction: %w", err)
	}
	return nil
}

func (db DBStorage) WriteURL(sd *ShortenerData) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(context.Background(),
		"INSERT INTO url_storage (short_url, original_url) VALUES($1,$2)", sd.ShortURL, sd.OriginalURL)
	if err != nil {
		// если ошибка, то откатываем изменения
		tx.Rollback()
		return err
	}
	// завершаем транзакцию
	return tx.Commit()
}

func (db DBStorage) ReadURL(URL string) (*ShortenerData, bool) {
	var origURL string
	db.db.QueryRow(
		"SELECT original_url FROM url_storage where short_url =$1", URL).Scan(&origURL)
	// готовим переменную для чтения результата
	if origURL != "" {
		log.Println("orig_url not null")
		log.Println(origURL)
		return &ShortenerData{
			OriginalURL: origURL,
		}, true
	} else {
		return nil, false
	}

}

func (db DBStorage) GetShortURLByOriginal(URL string) (*ShortenerData, bool) {
	var shortURL string
	db.db.QueryRow(
		"SELECT short_url FROM url_storage where original_url =$1", URL).Scan(&shortURL)
	// готовим переменную для чтения результата
	if shortURL != "" {
		return &ShortenerData{
			ShortURL: shortURL,
		}, true
	} else {
		return nil, false
	}
}