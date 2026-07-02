package serviceaccounts

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDomainRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "production is ranked first",
			domain: "production",
			want:   "000",
		},
		{
			name:   "development is ranked second",
			domain: "development",
			want:   "001",
		},
		{
			name:   "staging is ranked third",
			domain: "staging",
			want:   "002",
		},
		{
			name:   "unknown domain gets fallback prefix and name",
			domain: "alpha-custom",
			want:   "003alpha-custom",
		},
		{
			name:   "empty string treated as unknown domain",
			domain: "",
			want:   "003",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := domainRank(tt.domain)
			if got != tt.want {
				t.Errorf("domainRank(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestGroupServiceAccounts(t *testing.T) {
	t.Parallel()

	// sa builds a ServiceAccount with all fields populated.
	sa := func(k8sName, project, domain string) ServiceAccount {
		return ServiceAccount{
			K8sServiceAccount:    k8sName,
			GoogleServiceAccount: k8sName + "@project.iam.gserviceaccount.com",
			UnionProject:         project,
			UnionDomain:          domain,
		}
	}

	// domain builds a DomainGroup from a name and a variadic list of service accounts.
	domain := func(name string, sas ...ServiceAccount) DomainGroup {
		return DomainGroup{Name: name, ServiceAccounts: sas}
	}

	// project builds a ProjectGroup from a name and a variadic list of domain groups.
	project := func(name string, domains ...DomainGroup) ProjectGroup {
		return ProjectGroup{Name: name, Domains: domains}
	}

	// data builds a ServiceAccountsData from a variadic list of project groups.
	data := func(projects ...ProjectGroup) ServiceAccountsData {
		return ServiceAccountsData{Projects: projects}
	}

	tests := []struct {
		name     string
		accounts []ServiceAccount
		want     ServiceAccountsData
	}{
		{
			name:     "nil input returns empty projects slice",
			accounts: nil,
			want:     data(),
		},
		{
			name:     "single service account",
			accounts: []ServiceAccount{sa("sa-1", "myproject", "production")},
			want: data(
				project("myproject",
					domain("production", sa("sa-1", "myproject", "production")),
				),
			),
		},
		{
			name: "multiple projects are sorted alphabetically",
			accounts: []ServiceAccount{
				sa("sa-1", "zebra", "production"),
				sa("sa-2", "alpha", "production"),
			},
			want: data(
				project("alpha",
					domain("production", sa("sa-2", "alpha", "production")),
				),
				project("zebra",
					domain("production", sa("sa-1", "zebra", "production")),
				),
			),
		},
		{
			name: "domains sorted by priority: production, development, staging",
			accounts: []ServiceAccount{
				sa("sa-staging", "myproject", "staging"),
				sa("sa-prod", "myproject", "production"),
				sa("sa-dev", "myproject", "development"),
			},
			want: data(
				project("myproject",
					domain("production", sa("sa-prod", "myproject", "production")),
					domain("development", sa("sa-dev", "myproject", "development")),
					domain("staging", sa("sa-staging", "myproject", "staging")),
				),
			),
		},
		{
			name: "unknown domains sort after known ones, then alphabetically",
			accounts: []ServiceAccount{
				sa("sa-zzz", "myproject", "zzz-env"),
				sa("sa-prod", "myproject", "production"),
				sa("sa-aaa", "myproject", "aaa-env"),
			},
			want: data(
				project("myproject",
					domain("production", sa("sa-prod", "myproject", "production")),
					domain("aaa-env", sa("sa-aaa", "myproject", "aaa-env")),
					domain("zzz-env", sa("sa-zzz", "myproject", "zzz-env")),
				),
			),
		},
		{
			name: "service accounts within a domain sorted by name",
			accounts: []ServiceAccount{
				sa("sa-z", "myproject", "production"),
				sa("sa-a", "myproject", "production"),
				sa("sa-m", "myproject", "production"),
			},
			want: data(
				project("myproject",
					domain("production",
						sa("sa-a", "myproject", "production"),
						sa("sa-m", "myproject", "production"),
						sa("sa-z", "myproject", "production"),
					),
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := groupServiceAccounts(tt.accounts)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("groupServiceAccounts() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
