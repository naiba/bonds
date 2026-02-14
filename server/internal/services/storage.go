package services

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	ErrStorageFileNotFound = errors.New("storage file not found")
	ErrStorageSaveFailed   = errors.New("storage save failed")
)

type Storage interface {
	Save(filename string, data io.Reader) (storedPath string, err error)
	Get(storedPath string) (io.ReadCloser, error)
	Delete(storedPath string) error
	URL(storedPath string) string
}

type LocalStorage struct {
	uploadDir string
}

func NewLocalStorage(uploadDir string) *LocalStorage {
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		panic(fmt.Sprintf("failed to create upload directory %s: %v", uploadDir, err))
	}
	return &LocalStorage{uploadDir: uploadDir}
}

func (s *LocalStorage) Save(filename string, data io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	datePath := time.Now().Format("2006/01/02")
	storedName := uuid.New().String() + ext
	storedPath := filepath.Join(datePath, storedName)

	fullDir := filepath.Join(s.uploadDir, datePath)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageSaveFailed, err)
	}

	fullPath := filepath.Join(s.uploadDir, storedPath)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageSaveFailed, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageSaveFailed, err)
	}

	return storedPath, nil
}

func (s *LocalStorage) Get(storedPath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.uploadDir, storedPath)
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrStorageFileNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *LocalStorage) Delete(storedPath string) error {
	fullPath := filepath.Join(s.uploadDir, storedPath)
	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrStorageFileNotFound
		}
		return err
	}
	return nil
}

func (s *LocalStorage) URL(storedPath string) string {
	return "/api/files/" + storedPath
}
