package handlers

import (
	"fmt"
	"net/http"
	"os/exec"

	"github.com/navikt/union-api/pkg/middleware"
)

func ServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	assignments, err := getUnionIdentityAssignments(principal)
	if err != nil {
		http.Error(w, "failed to fetch identity assignments", http.StatusInternalServerError)
		return
	}

	_ = assignments // TODO: render response
}

func getUnionIdentityAssignments(principal *middleware.Principal) ([]string, error) {
	cmd := exec.Command(
		"uctl",
		"--org", "union-nav",
		"get", "identityassignments",
		"--user", principal.Email,
		"--output", "json",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute uctl command: %w", err)
	}
	_ = output // TODO: parse JSON output
	return []string{"assignment1", "assignment2"}, nil
}
