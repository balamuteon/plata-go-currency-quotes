package quote

import "errors"

var (
	ErrInvalidJSON               = errors.New("invalid json")
	ErrRequestBodyTooLarge       = errors.New("request body is too large")
	ErrQuoteUpdateNotFound       = errors.New("quote update was not found")
	ErrQuoteNotFound             = errors.New("quote was not found")
	ErrStatusInternalServerError = errors.New("internal server error")
)
