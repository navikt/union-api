package middleware

import (
	"context"
	"net/http"
)

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := &Principal{
			Email: "foo.bar@nav.no", 
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, "principal", principal)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

type Principal struct {
	Email string	
}
