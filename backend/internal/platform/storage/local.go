package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

func (l *LocalStorage) Put(_ context.Context, in PutObjectInput) (PutObjectOutput, error) {
	fullPath, err := l.resolvePath(in.ObjectKey)
	if err != nil {
		return PutObjectOutput{}, err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
		return PutObjectOutput{}, err
	}

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return PutObjectOutput{}, err
	}
	defer f.Close()

	if _, err := io.Copy(f, in.Body); err != nil {
		return PutObjectOutput{}, err
	}

	return PutObjectOutput{
		Provider:  "local",
		ObjectKey: in.ObjectKey,
	}, nil
}

func (l *LocalStorage) Open(_ context.Context, in GetObjectInput) (io.ReadCloser, error) {
	fullPath, err := l.resolvePath(in.ObjectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

func (l *LocalStorage) Delete(_ context.Context, in GetObjectInput) error {
	fullPath, err := l.resolvePath(in.ObjectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

func (l *LocalStorage) resolvePath(objectKey string) (string, error) {
	cleanKey := filepath.Clean(strings.TrimPrefix(objectKey, "/"))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") {
		return "", errors.New("invalid object key")
	}

	fullPath := filepath.Join(l.basePath, cleanKey)
	baseClean := filepath.Clean(l.basePath)
	fullClean := filepath.Clean(fullPath)
	if fullClean != baseClean && !strings.HasPrefix(fullClean, baseClean+string(os.PathSeparator)) {
		return "", errors.New("object key escapes base path")
	}
	return fullClean, nil
}
