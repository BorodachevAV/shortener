package data_base

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/BorodachevAV/shortener/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
)

type DBStorage struct {
	db  *sql.DB
	ctx context.Context
}

func NewDBStorage(DNS string, ctx context.Context) (*DBStorage, error) {
	db, err := sql.Open("pgx", DNS)
	if err != nil {
		log.Println(err.Error())
	}
	storage := &DBStorage{
		db:  db,
		ctx: ctx,
	}
	err = storage.CreateSchema()
	if err != nil {
		log.Println(err.Error())
	}
	return storage, nil
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

func (db DBStorage) WriteURL(sd *storage.ShortenerData) error {
	isDuplicate, _ := db.CheckDuplicateURL(sd.OriginalURL)
	if isDuplicate != "" {
		return storage.ErrDuplicate
	}
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

func (db DBStorage) WriteBatch(sd []*storage.ShortenerData) error {
	for _, data := range sd {
		err := db.WriteURL(data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db DBStorage) ReadURL(URL string) (*storage.ShortenerData, error) {
	var origURL string
	db.db.QueryRow(
		"SELECT original_url FROM url_storage where short_url =$1", URL).Scan(&origURL)
	// готовим переменную для чтения результата
	if origURL != "" {
		log.Println("orig_url not null")
		log.Println(origURL)
		return &storage.ShortenerData{
			OriginalURL: origURL,
		}, nil
	} else {
		return nil, nil
	}

}

func (db DBStorage) GetShortUrlByOriginalURL(URL string) (*storage.ShortenerData, error) {
	var shortURL string
	db.db.QueryRow(
		"SELECT short_url FROM url_storage where original_url =$1", URL).Scan(&shortURL)
	// готовим переменную для чтения результата
	if shortURL != "" {
		return &storage.ShortenerData{
			ShortURL: shortURL,
		}, nil
	} else {
		return nil, nil
	}
}

func (db DBStorage) CheckDuplicateURL(originalURL string) (string, error) {
	var shortURL string
	db.db.QueryRow(
		"SELECT short_url FROM url_storage where original_url =$1", originalURL).Scan(&shortURL)
	// готовим переменную для чтения результата
	if shortURL != "" {
		return shortURL, nil
	} else {
		return "", nil
	}
}
