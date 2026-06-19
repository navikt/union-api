package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/navikt/union-api/pkg/k8s"
	"github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/pkg/uctl"
)

type ServiceAccountsHandler struct {
	uctlClient uctl.UCTLClient
	k8sClient  *k8s.K8sClient
}

func NewServiceAccountsHandler(uctlClient uctl.UCTLClient, k8sClient *k8s.K8sClient) ServiceAccountsHandler {
	return ServiceAccountsHandler{
		uctlClient,
		k8sClient,
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

	ns, err := h.k8sClient.Namespaces(r.Context())
	if err != nil {
		slog.Error("failed to get k8s server version", "error", err)
		http.Error(w, "failed to get k8s server version", http.StatusInternalServerError)
		return
	}

	slog.Info("Namespaces:")
	for _, ns := range ns.Items {
		slog.Info(ns.Name)
	}

	json, err := json.Marshal(assignments)
	w.Write(json)
}
