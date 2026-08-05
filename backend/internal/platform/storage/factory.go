package storage

import "pekan/backend/internal/platform/config"

func NewProvider(cfg config.Config) ObjectStorage {
	switch cfg.StorageProvider {
	case "s3":
		provider, err := NewS3Storage(cfg.StorageS3Region, cfg.StorageS3Bucket, cfg.StorageS3AccessKey, cfg.StorageS3SecretKey, cfg.StorageS3Endpoint)
		if err != nil {
			return NewLocalStorage(cfg.StorageLocalPath) // Fallback
		}
		return provider
	case "gdrive":
		provider, err := NewGoogleDriveStorage(cfg.StorageDriveFolder, cfg.StorageGDriveCredentials)
		if err != nil {
			return NewLocalStorage(cfg.StorageLocalPath) // Fallback
		}
		return provider
	default:
		return NewLocalStorage(cfg.StorageLocalPath)
	}
}

