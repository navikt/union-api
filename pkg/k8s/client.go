package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

type K8sClient struct {
	clientset *kubernetes.Clientset
}

func NewK8sClient(ctx context.Context, cfg K8sConfig) (*K8sClient, error) {
	ts, err := google.DefaultTokenSource(ctx, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("get default token source: %w", err)
	}

	restCfg := &rest.Config{Host: cfg.host()}
	restCfg.Wrap(
		func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{
				Source: ts,
				Base: &gatewayTransport{
					base:       rt,
					pathPrefix: cfg.pathPrefix(),
				},
			}
		},
	)

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	return &K8sClient{clientset: clientset}, nil
}

func (c *K8sClient) Namespaces(ctx context.Context) (*corev1.NamespaceList, error) {
	ns, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	return ns, nil
}

// gatewayTransport rewrites every request URL to prepend the Connect Gateway
// path prefix. This is necessary because client-go builds request paths from
// versionedAPIPath (e.g. /api/v1) without preserving any path component set
// on rest.Config.Host, so the prefix must be injected at the transport level.
type gatewayTransport struct {
	base       http.RoundTripper
	pathPrefix string
}

func (t *gatewayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r2 := req.Clone(req.Context())
	r2.URL.Path = t.pathPrefix + req.URL.Path
	if req.URL.RawPath != "" {
		r2.URL.RawPath = t.pathPrefix + req.URL.RawPath
	}
	slog.Debug("k8s request", "before", req.URL.String(), "after", r2.URL.String())
	return t.base.RoundTrip(r2)
}
