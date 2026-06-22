package serviceaccounts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/union-api/pkg/k8s"
	"github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/pkg/uctl"
	corev1 "k8s.io/api/core/v1"
)

type Service struct {
	uctlClient uctl.UCTLClient
	k8sClient  *k8s.K8sClient
}

func NewService(uctlClient uctl.UCTLClient, k8sClient *k8s.K8sClient) Service {
	return Service{
		uctlClient,
		k8sClient,
	}
}

func (s Service) GetServiceAccounts(ctx context.Context, principal *middleware.Principal) ([]ServiceAccount, error) {
	permissions, err := s.uctlClient.GetIdentityAssignments(principal.Email)
	if err != nil {
		slog.Error("failed to fetch identity assignments", "error", err)
		return nil, fmt.Errorf("failed to fetch identity assignments")
	}

	var serviceAccounts []ServiceAccount

	for _, resource := range projectResources(permissions) {
		ns, err := resource.Namespace()
		if err != nil {
			return nil, err
		}

		k8sServiceAccounts, err := s.k8sClient.ServiceAccounts(ctx, ns)
		if err != nil {
			slog.Error("failed to fetch kubernetes service accounts", "error", err)
			return nil, fmt.Errorf("failed to fetch kubernetes service accounts")
		}
		for _, k8sSa := range k8sServiceAccounts.Items {
			if sa, ok := toServiceAccount(k8sSa, resource); ok {
				serviceAccounts = append(serviceAccounts, sa)
			}
		}
	}

	return serviceAccounts, nil
}

func projectResources(permissions []uctl.Permission) []uctl.Resource {
	var resources []uctl.Resource
	for _, permission := range permissions {
		for _, resource := range permission.Resources {
			if resource.Kind == uctl.Project {
				resources = append(resources, resource)
			}
		}
	}
	return resources
}

func toServiceAccount(k8sSa corev1.ServiceAccount, resource uctl.Resource) (ServiceAccount, bool) {
	if k8sSa.Name == "default" {
		return ServiceAccount{}, false
	}
	gsa, ok := k8sSa.Annotations["iam.gke.io/gcp-service-account"]
	if !ok {
		return ServiceAccount{}, false
	}
	return ServiceAccount{
		K8sServiceAccount:    k8sSa.Name,
		GoogleServiceAccount: gsa,
		UnionProject:         resource.Project,
		UnionDomain:          resource.Domain,
	}, true
}
