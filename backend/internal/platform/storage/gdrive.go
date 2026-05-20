package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleDriveStorage struct {
	service  *drive.Service
	folderID string
}

func NewGoogleDriveStorage(folderID, credentialsJSON string) (*GoogleDriveStorage, error) {
	if credentialsJSON == "" {
		return nil, errors.New("google drive credentials not provided")
	}

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsJSON([]byte(credentialsJSON)), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &GoogleDriveStorage{
		service:  srv,
		folderID: folderID,
	}, nil
}

func (s *GoogleDriveStorage) Put(ctx context.Context, in PutObjectInput) (PutObjectOutput, error) {
	f := &drive.File{
		Name:    in.ObjectKey,
		Parents: []string{s.folderID},
	}
	
	_, err := s.service.Files.Create(f).Media(in.Body).Do()
	if err != nil {
		return PutObjectOutput{}, fmt.Errorf("gdrive upload failed: %w", err)
	}

	return PutObjectOutput{
		Provider:  "gdrive",
		ObjectKey: in.ObjectKey,
	}, nil
}

func (s *GoogleDriveStorage) Open(ctx context.Context, in GetObjectInput) (io.ReadCloser, error) {
	// Note: GDrive search by name/path is complex. 
	// This placeholder assumes we have the FileID or can find it by Name in the folder.
	return nil, errors.New("gdrive open not implemented for general object keys")
}

func (s *GoogleDriveStorage) Delete(ctx context.Context, in GetObjectInput) error {
	return errors.New("gdrive delete not implemented")
}
