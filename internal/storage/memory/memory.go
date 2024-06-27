package memory

import (
	"github.com/BorodachevAV/shortener/internal/storage"
	"sync"
)

type MapStorage struct {
	UrlsStorage *sync.Map
}

func (f MapStorage) WriteURL(data *storage.ShortenerData) error {
	f.UrlsStorage.Store(data.ShortURL, data.OriginalURL)
	return nil
}

func (f MapStorage) WriteBatch(data []*storage.ShortenerData) error {
	for _, sd := range data {
		err := f.WriteURL(sd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f MapStorage) ReadURL(URL string) (*storage.ShortenerData, error) {
	val, ok := f.UrlsStorage.Load(URL)
	if !ok {
		return nil, nil
	}
	resp := storage.ShortenerData{
		OriginalURL: val.(string),
	}
	return &resp, nil
}

func (f MapStorage) CheckDuplicateURL(originalURL string) (string, error) {
	return "", nil
}

func (f MapStorage) GetUserURLs(userID string) ([]*storage.ShortenerData, error) {
	return nil, nil
}

func (f MapStorage) DeleteUserURLs([]*storage.ShortenerData) error {
	return nil
}
