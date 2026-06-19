package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/pkg/uctl"
)

type ServiceAccountsHandler struct {
	uctlClient uctl.UCTLClient
}

func NewServiceAccountsHandler(uctlClient uctl.UCTLClient) ServiceAccountsHandler {
	return ServiceAccountsHandler{
		uctlClient,
	}
}

func (h ServiceAccountsHandler) GetServiceAccounts(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	assignments, err := h.uctlClient.GetIdentityAssignments(principal.Email)
	if err != nil {
		slog.Error("failed to fetch identity assignments", "error", err)
		http.Error(w, "failed to fetch identity assignments", http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(assignments)
	w.Write(json)
}
