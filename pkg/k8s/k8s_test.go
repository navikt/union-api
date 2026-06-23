package k8s

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes"
)

func TestConnectGatewayURL(t *testing.T) {
	cfg := GKEConfig{
		FleetHostProjectNumber: "123456789",
		Location:               "europe-west1",
		MembershipName:         "dev-union-restricted",
	}

	got := cfg.ConnectGatewayURL()
	want := "https://europe-west1-connectgateway.googleapis.com/v1/projects/123456789/locations/europe-west1/gkeMemberships/dev-union-restricted"
	if got != want {
		t.Errorf("ConnectGatewayURL() = %q, want %q", got, want)
	}
}

// TestGatewayPathAndAuth guards the two behaviours that let us drop the custom
// gatewayTransport: client-go prepends the host's path to every request, and the
// oauth2.Transport wrap attaches the bearer token. If either regresses (e.g. by
// reintroducing manual path rewriting), this test fails.
func TestGatewayPathAndAuth(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"kind":"NamespaceList","apiVersion":"v1","metadata":{},"items":[]}`)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
	host := srv.URL + "/v1/projects/p/locations/l/gkeMemberships/m"

	clientset, err := kubernetes.NewForConfig(newRestConfig(host, ts))
	if err != nil {
		t.Fatalf("NewForConfig: %v", err)
	}
	client := &K8sClient{clientset: clientset}

	if _, err := client.Namespaces(context.Background()); err != nil {
		t.Fatalf("Namespaces: %v", err)
	}

	wantPath := "/v1/projects/p/locations/l/gkeMemberships/m/api/v1/namespaces"
	if gotPath != wantPath {
		t.Errorf("request path = %q, want %q", gotPath, wantPath)
	}
	if want := "Bearer test-token"; gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
}
