package server

import (
	"net/http"
)

func NewServer(handler http.Handler) (*http.Server, error) {
	return &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}, nil
}
