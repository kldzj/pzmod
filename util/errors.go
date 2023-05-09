package util

import (
	"errors"
)

var (
	ErrFileNotFound = errors.New("file not found")
	ErrFileExists   = errors.New("file already exists")
	ErrInvalidKey   = errors.New("invalid key")
	ErrVerNotSet    = errors.New("version not set")
	ErrNoApiKey     = errors.New("no Steam API key set")
	ErrNoCredFile   = errors.New("no credentials file found")
)
