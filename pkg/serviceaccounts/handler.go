package serviceaccounts

import (
	"encoding/json"
	"net/http"

	"github.com/navikt/union-api/pkg/middleware"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service}
}

func (h Handler) GetServiceAccounts(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accounts, err := h.service.GetServiceAccounts(r.Context(), principal)
	if err != nil {
		http.Error(w, "Unable to fetch serviceaccounts", http.StatusInternalServerError)
	}

	json, err := json.Marshal(accounts)
	w.Write(json)
}
