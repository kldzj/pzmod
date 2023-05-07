package util

import (
	"errors"
)

var (
	ErrFileNotFound = errors.New("File not found")
	ErrFileExists   = errors.New("File already exists")
	ErrInvalidKey   = errors.New("Invalid key")
)
