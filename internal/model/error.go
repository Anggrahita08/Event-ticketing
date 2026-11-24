package model

import "errors"

var (
	ErrNotFound     = errors.New("data tidak ditemukan")
	ErrConflict     = errors.New("data duplikat")
	ErrUnauthorized = errors.New("email atau password salah")
)
