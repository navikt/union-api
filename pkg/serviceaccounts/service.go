package serviceaccounts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/union-api/pkg/k8s"
	"github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/pkg/uctl"
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

	for _, permission := range permissions {
		for _, resource := range permission.Resources {
			if resource.Kind != uctl.Project {
				continue
			}
			ns, err := resource.Namespace()
			if err != nil {
				return nil, err
			}

			k8sServiceAccounts, err := s.k8sClient.ServiceAccounts(ctx, ns)
			if err != nil {
				slog.Error("faild to fetch kubernetes service accounts", "error", err)
				return nil, fmt.Errorf("faild to fetch kubernetes service accounts")
			}
			for _, sa := range k8sServiceAccounts.Items {
				if sa.Name == "default" {
					continue
				}
				gsa, ok := sa.Annotations["iam.gke.io/gcp-service-account"]
				if !ok {
					continue
				}
				serviceAccounts = append(serviceAccounts, ServiceAccount{
					K8sServiceAccount:    sa.Name,
					GoogleServiceAccount: gsa,
					UnionProject:         resource.Project,
					UnionDomain:          resource.Domain,
				})
			}
		}
	}

	return serviceAccounts, nil
}
