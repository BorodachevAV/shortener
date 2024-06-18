package file

import (
	"bufio"
	"encoding/json"
	"github.com/BorodachevAV/shortener/internal/storage"
	"log"
	"os"
)

type FileStorage struct {
	filename string
	file     *os.File
	scanner  *bufio.Scanner
}

func NewFileStorage(filename string) (*FileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		filename: filename,
		file:     file,
		scanner:  bufio.NewScanner(file)}, nil
}

func (f FileStorage) WriteURL(sd *storage.ShortenerData) error {
	data, _ := f.ReadURL(sd.ShortURL)
	var newData storage.ShortenerData
	if data != nil {
		newData.ID = data.ID + 1
		newData.ShortURL = sd.ShortURL
		newData.OriginalURL = sd.OriginalURL

	} else {
		newData.ID = 1
		newData.ShortURL = sd.ShortURL
		newData.OriginalURL = sd.OriginalURL
	}
	result, err := json.Marshal(&newData)
	if err != nil {
		return err
	}
	result = append(result, '\n')
	log.Println(string(result))
	file, err := os.OpenFile(f.filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	_, err = file.Write(result)
	file.Close()
	return err
}

func (f FileStorage) ReadURL(shortURL string) (*storage.ShortenerData, error) {
	file, err := os.OpenFile(f.filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		log.Println("empty shortener")
		return nil, nil
	}
	data := scanner.Bytes()
	storageData := storage.ShortenerData{}
	err = json.Unmarshal(data, &storageData)
	log.Println(&storageData)
	if err != nil {
		return nil, err
	}
	file.Close()

	return &storageData, nil
}

func (f FileStorage) WriteBatch(data []*storage.ShortenerData) error {
	for _, sd := range data {
		err := f.WriteURL(sd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f FileStorage) CheckDuplicateURL(originalURL string) (string, error) {
	return "", nil
}
