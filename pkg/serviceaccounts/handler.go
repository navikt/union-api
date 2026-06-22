package serviceaccounts

import (
	"net/http"

	"github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/web"
)

type Handler struct {
	service  Service
	renderer *web.Renderer
}

func NewHandler(service Service, renderer *web.Renderer) Handler {
	return Handler{service, renderer}
}

func (h Handler) GetServiceAccounts(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accounts, err := h.service.GetServiceAccounts(r.Context(), principal)
	if err != nil {
		http.Error(w, "Unable to fetch service accounts", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, "serviceaccounts", principal, groupServiceAccounts(accounts))
}
