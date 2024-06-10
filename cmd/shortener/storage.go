package main

import (
	"bufio"
	"encoding/json"
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
	urlsStorage *sync.Map
}

func (f MapStorage) WriteURL(data *ShortenerData) error {
	f.urlsStorage.Store(data.ShortURL, data.OriginalURL)
	return nil
}

func (f MapStorage) ReadURL(URL string) (*ShortenerData, bool) {
	val, ok := f.urlsStorage.Load(URL)
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

func (f FileStorage) Delete() error {
	// закрываем файл
	return f.file.Close()
}

type StorageReader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewStorageReader(filename string) (*StorageReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &StorageReader{file: file,
		scanner: bufio.NewScanner(file),
	}, nil
}

func (r *StorageReader) ReadData() (*ShortenerData, bool) {
	if !r.scanner.Scan() {
		return nil, false
	}
	data := r.scanner.Bytes()

	storageData := ShortenerData{}
	err := json.Unmarshal(data, &storageData)
	if err != nil {
		return nil, false
	}

	return &storageData, true
}

func (r *StorageReader) Close() error {
	// закрываем файл
	return r.file.Close()
}
