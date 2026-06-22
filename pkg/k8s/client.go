package k8s

import (
	"context"
	"fmt"
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

	clientset, err := kubernetes.NewForConfig(newRestConfig(cfg.ConnectGatewayURL(), ts))
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	return &K8sClient{clientset: clientset}, nil
}

// newRestConfig builds a rest.Config that talks to the Connect Gateway at host
// and authenticates with a GCP OAuth2 token from ts.
//
// host carries the Connect Gateway membership path; client-go applies that path
// as a prefix to every request URL, so no custom RoundTripper is needed. The
// oauth2.Transport wrap injects (and refreshes) the bearer token, replacing the
// in-tree gcp auth provider that client-go removed in v1.26.
func newRestConfig(host string, ts oauth2.TokenSource) *rest.Config {
	restCfg := &rest.Config{Host: host}
	restCfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{Source: ts, Base: rt}
	})
	return restCfg
}

func (c *K8sClient) Namespaces(ctx context.Context) (*corev1.NamespaceList, error) {
	ns, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	return ns, nil
}

func (c *K8sClient) ServiceAccounts(ctx context.Context, namespace string) (*corev1.ServiceAccountList, error) {
	serviceAccounts, err := c.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list serviceaccounts: %w", err)
	}

	return serviceAccounts, nil
}
