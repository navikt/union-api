// Package health provides liveness and readiness HTTP handlers for Kubernetes
// probes. The endpoints are intentionally unauthenticated and must be mounted
// outside the session middleware so the kubelet can reach them.
package health

import "net/http"

// Alive reports that the process is running. It must stay cheap and dependency
// free: a failing liveness probe causes Kubernetes to restart the pod.
func Alive(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// Ready reports that the service can serve traffic. It currently returns 200
// unconditionally; once the process is up its dependencies (k8s client, uctl)
// are configured at startup. Extend this to probe the Kubernetes API if we want
// readiness to gate on upstream availability.
func Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
