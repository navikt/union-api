package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/pkg/serviceaccounts"
)

type ServiceAccountsHandler struct {
	serviceAccountService serviceaccounts.Service
}

func NewServiceAccountsHandler(serviceAccountService serviceaccounts.Service) ServiceAccountsHandler {
	return ServiceAccountsHandler{
		serviceAccountService,
	}
}

func (h ServiceAccountsHandler) GetServiceAccounts(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	serviceaccounts, err := h.serviceAccountService.GetServiceAccounts(r.Context(), principal)
	if err != nil {
		http.Error(w, "Unable to fetch serviceaccounts", http.StatusInternalServerError)
	}

	json, err := json.Marshal(serviceaccounts)
	w.Write(json)
}
