package server

import "net/http"
import "github.com/navikt/union-api/pkg/routes"

func NewServer() *http.Server {
	r := routes.ServiceAccountsRouter()

	return &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
}
