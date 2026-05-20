package domain

import "errors"

var (
	ErrInvalidAccountName   = errors.New("invalid account name")
	ErrInvalidAccountType   = errors.New("invalid account type")
	ErrInvalidCurrency      = errors.New("invalid currency")
	ErrInvalidCategoryName  = errors.New("invalid category name")
	ErrInvalidCategoryType  = errors.New("invalid category type")
)

