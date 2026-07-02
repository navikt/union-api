package uctl

import (
	"strings"
	"testing"
)

func TestResource_Namespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource Resource
		want     string
		wantErr  string
	}{
		{
			name:     "project resource returns project-domain namespace",
			resource: Resource{Kind: Project, Organization: "org", Domain: "production", Project: "myproject"},
			want:     "myproject-production",
		},
		{
			name:     "organization resource returns error",
			resource: Resource{Kind: Organization, Organization: "org"},
			wantErr:  "organization",
		},
		{
			name:     "domain resource returns error",
			resource: Resource{Kind: Domain, Organization: "org", Domain: "production"},
			wantErr:  "domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.resource.Namespace()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Namespace() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Namespace() error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Namespace() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Namespace() = %q, want %q", got, tt.want)
			}
		})
	}
}
