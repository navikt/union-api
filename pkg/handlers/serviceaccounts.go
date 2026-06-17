package handlers

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/navikt/union-api/pkg/middleware"
)

func ServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	principal, ok := ctx.Value("principal").(*middleware.Principal)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	

}

func getUnionIdentityAssignments(principal *middleware.Principal) ([]string, error) {
	cmd := exec.Command("uctl", "--org", "union-nav", "get", "identityassignments", "--user", principal.Email, "--output", "json")	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute uctl command: %w", err)
	}
	return []string{"assignment1", "assignment2"}, nil
}