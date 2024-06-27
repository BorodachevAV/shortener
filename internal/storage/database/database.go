package database

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

	createShema :=
		`CREATE TABLE IF NOT EXISTS url_storage(
			short_url VARCHAR(200) PRIMARY KEY,
			original_url VARCHAR(200) NOT NULL UNIQUE,
    		user_id VARCHAR(200),
    		deleted_flag BOOLEAN NOT NULL DEFAULT FALSE
		)`

	_, err := db.db.Query(createShema)
	return err
}

func (db DBStorage) WriteURL(sd *storage.ShortenerData) error {
	isDuplicate, _ := db.CheckDuplicateURL(sd.OriginalURL)
	if isDuplicate != "" {
		return storage.ErrDuplicate
	}

	_, err := db.db.Query("INSERT INTO url_storage (short_url, original_url, user_id) VALUES($1,$2,$3)", sd.ShortURL, sd.OriginalURL, sd.UserID)
	return err
}

func (db DBStorage) WriteBatch(sd []*storage.ShortenerData) error {
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
	stmt, err := tx.Prepare("INSERT INTO url_storage (short_url, original_url, user_id) VALUES($1,$2,$3)")
	if err != nil {
		log.Fatal(err)
	}
	for _, sd := range sd {
		if _, err := stmt.Exec(sd.ShortURL, sd.OriginalURL, sd.UserID); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the transaction: %w", err)
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

func (db DBStorage) GetUserURLs(userID string) ([]*storage.ShortenerData, error) {
	var results []*storage.ShortenerData
	rows, err := db.db.Query(
		"SELECT short_url, original_url FROM url_storage where user_id =$1", userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		sd := &storage.ShortenerData{}
		rows.Scan(&sd.ShortURL, &sd.OriginalURL)
		results = append(results, sd)
	}
	return results, nil
}

func (db DBStorage) DeleteUserURLs(sd []*storage.ShortenerData) error {
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
	stmt, err := tx.Prepare("UPDATE url_storage SET deleted_flag = TRUE where (short_url=$1 and user_id=$2)")
	if err != nil {
		log.Fatal(err)
	}
	for _, sd := range sd {
		if _, err := stmt.Exec(sd.ShortURL, sd.UserID); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the transaction: %w", err)
	}
	return nil
}
