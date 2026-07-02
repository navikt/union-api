package serviceaccounts

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/union-api/pkg/uctl"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProjectResources(t *testing.T) {
	t.Parallel()

	proj := func(org, domain, project string) uctl.Resource {
		return uctl.Resource{Kind: uctl.Project, Organization: org, Domain: domain, Project: project}
	}

	tests := []struct {
		name        string
		permissions []uctl.Permission
		want        []uctl.Resource
	}{
		{
			name:        "nil input returns nil",
			permissions: nil,
			want:        nil,
		},
		{
			name:        "empty input returns nil",
			permissions: []uctl.Permission{},
			want:        nil,
		},
		{
			name: "non-project resources are excluded",
			permissions: []uctl.Permission{
				{Resources: []uctl.Resource{
					{Kind: uctl.Organization, Organization: "org"},
					{Kind: uctl.Domain, Organization: "org", Domain: "production"},
				}},
			},
			want: nil,
		},
		{
			name: "only project resources are returned",
			permissions: []uctl.Permission{
				{Resources: []uctl.Resource{
					proj("org", "production", "alpha"),
					proj("org", "development", "alpha"),
				}},
			},
			want: []uctl.Resource{
				proj("org", "production", "alpha"),
				proj("org", "development", "alpha"),
			},
		},
		{
			name: "project resources are extracted from mixed kinds",
			permissions: []uctl.Permission{
				{Resources: []uctl.Resource{
					{Kind: uctl.Organization, Organization: "org"},
					proj("org", "production", "alpha"),
					{Kind: uctl.Domain, Organization: "org", Domain: "production"},
					proj("org", "staging", "beta"),
				}},
			},
			want: []uctl.Resource{
				proj("org", "production", "alpha"),
				proj("org", "staging", "beta"),
			},
		},
		{
			name: "project resources are collected across multiple permissions",
			permissions: []uctl.Permission{
				{Resources: []uctl.Resource{proj("org", "production", "alpha")}},
				{Resources: []uctl.Resource{proj("org", "development", "beta")}},
			},
			want: []uctl.Resource{
				proj("org", "production", "alpha"),
				proj("org", "development", "beta"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := projectResources(tt.permissions)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("projectResources() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToServiceAccount(t *testing.T) {
	t.Parallel()

	const gkeAnnotation = "iam.gke.io/gcp-service-account"

	resource := uctl.Resource{
		Kind:         uctl.Project,
		Organization: "org",
		Domain:       "production",
		Project:      "myproject",
	}

	k8sSA := func(name string, annotations map[string]string) corev1.ServiceAccount {
		return corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: annotations,
			},
		}
	}

	tests := []struct {
		name   string
		k8sSA  corev1.ServiceAccount
		want   ServiceAccount
		wantOk bool
	}{
		{
			name:   "default service account is excluded",
			k8sSA:  k8sSA("default", map[string]string{gkeAnnotation: "sa@project.iam.gserviceaccount.com"}),
			want:   ServiceAccount{},
			wantOk: false,
		},
		{
			name:   "service account without GKE annotation is excluded",
			k8sSA:  k8sSA("my-sa", nil),
			want:   ServiceAccount{},
			wantOk: false,
		},
		{
			name:  "valid service account with annotation is included",
			k8sSA: k8sSA("my-sa", map[string]string{gkeAnnotation: "my-sa@project.iam.gserviceaccount.com"}),
			want: ServiceAccount{
				K8sServiceAccount:    "my-sa",
				GoogleServiceAccount: "my-sa@project.iam.gserviceaccount.com",
				UnionProject:         "myproject",
				UnionDomain:          "production",
			},
			wantOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := toServiceAccount(tt.k8sSA, resource)
			if ok != tt.wantOk {
				t.Fatalf("toServiceAccount() ok = %v, want %v", ok, tt.wantOk)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("toServiceAccount() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
