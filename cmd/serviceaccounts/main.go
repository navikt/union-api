package main

import "github.com/navikt/union-api/pkg/server"

func main() {
	server := server.NewServer()
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}