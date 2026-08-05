package domain

import "errors"

var (
	ErrProviderNotConfigured     = errors.New("receipt scan provider is not configured")
	ErrProviderCredentialInvalid = errors.New("saved API key cannot be decrypted")
	ErrInvalidProvider           = errors.New("invalid receipt scan provider")
	ErrInvalidFile               = errors.New("invalid receipt file")
	ErrNoConfiguredProvider      = errors.New("receipt scan is not configured")
)
