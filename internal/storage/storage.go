package storage

import "errors"

var ErrDuplicate = errors.New("duplicated originalUrl")

type ShortenerData struct {
	ID          uint   `json:"uuID"`
	UserID      string `json:"userID"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ShortenerStorage interface {
	WriteURL(*ShortenerData) error
	ReadURL(shortURL string) (*ShortenerData, error)
	WriteBatch([]*ShortenerData) error
	CheckDuplicateURL(originalURL string) (string, error)
	GetUserURLs(userID string) ([]*ShortenerData, error)
	DeleteUserURLs([]*ShortenerData) error
}
