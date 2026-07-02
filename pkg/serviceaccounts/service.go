package serviceaccounts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/union-api/pkg/auth"
	"github.com/navikt/union-api/pkg/k8s"
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

func (s Service) GetServiceAccounts(ctx context.Context, principal *auth.Principal) ([]ServiceAccount, error) {
	permissions, err := s.uctlClient.GetIdentityAssignments(ctx, principal.Email)
	if err != nil {
		slog.Error("failed to fetch identity assignments", "error", err)
		return nil, fmt.Errorf("failed to fetch identity assignments")
	}

	resources := projectResources(permissions)

	accountsByNS, err := s.fetchAllServiceAccounts(ctx, resources)
	if err != nil {
		return nil, err
	}

	return assembleServiceAccounts(resources, accountsByNS), nil
}

func (s Service) fetchAllServiceAccounts(ctx context.Context, resources []uctl.Resource) (map[string][]corev1.ServiceAccount, error) {
	accountsByNS := make(map[string][]corev1.ServiceAccount, len(resources))
	for _, resource := range resources {
		ns, err := resource.Namespace()
		if err != nil {
			return nil, err
		}
		k8sSAs, err := s.k8sClient.ServiceAccounts(ctx, ns)
		if err != nil {
			slog.Error("failed to fetch kubernetes service accounts", "error", err)
			return nil, fmt.Errorf("failed to fetch kubernetes service accounts")
		}
		accountsByNS[ns] = k8sSAs.Items
	}
	return accountsByNS, nil
}

func assembleServiceAccounts(resources []uctl.Resource, accountsByNS map[string][]corev1.ServiceAccount) []ServiceAccount {
	var serviceAccounts []ServiceAccount
	for _, resource := range resources {
		ns, _ := resource.Namespace() // namespace was already validated in fetchAllServiceAccounts
		for _, k8sSa := range accountsByNS[ns] {
			if sa, ok := toServiceAccount(k8sSa, resource); ok {
				serviceAccounts = append(serviceAccounts, sa)
			}
		}
	}
	return serviceAccounts
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
