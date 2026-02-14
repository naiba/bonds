package services

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorageSaveAndGet(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	content := "hello world"
	storedPath, err := storage.Save("test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if storedPath == "" {
		t.Fatal("Save returned empty storedPath")
	}

	if filepath.Ext(storedPath) != ".txt" {
		t.Fatalf("Expected .txt extension, got %s", filepath.Ext(storedPath))
	}

	reader, err := storage.Get(storedPath)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(data) != content {
		t.Fatalf("Expected %q, got %q", content, string(data))
	}
}

func TestLocalStorageDelete(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	storedPath, err := storage.Save("delete-me.txt", strings.NewReader("bye"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = storage.Delete(storedPath)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = storage.Get(storedPath)
	if !errors.Is(err, ErrStorageFileNotFound) {
		t.Fatalf("Expected ErrStorageFileNotFound after delete, got %v", err)
	}
}

func TestLocalStorageDeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	err := storage.Delete("nonexistent/file.txt")
	if !errors.Is(err, ErrStorageFileNotFound) {
		t.Fatalf("Expected ErrStorageFileNotFound, got %v", err)
	}
}

func TestLocalStorageGetNotFound(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	_, err := storage.Get("nonexistent/file.txt")
	if !errors.Is(err, ErrStorageFileNotFound) {
		t.Fatalf("Expected ErrStorageFileNotFound, got %v", err)
	}
}

func TestLocalStorageURL(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	url := storage.URL("2026/02/14/abc.jpg")
	expected := "/api/files/2026/02/14/abc.jpg"
	if url != expected {
		t.Fatalf("Expected %q, got %q", expected, url)
	}
}

func TestLocalStorageSaveNoExtension(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocalStorage(dir)

	storedPath, err := storage.Save("noext", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if filepath.Ext(storedPath) != "" {
		t.Fatalf("Expected no extension, got %s", filepath.Ext(storedPath))
	}
}
