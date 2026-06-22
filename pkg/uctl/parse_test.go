package uctl

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParsePermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []Permission
		wantErr bool
	}{
		{
			name: "happy path with preamble and multi-resource entry",
			input: `No union block in specified .yaml file. Defaulting to admin block instead.
[
	{
		"Name": "admin",
		"Role": "admin",
		"Resource": "organization [union-nav]"
	},
	{
		"Name": "dataplattform-project-owners",
		"Role": "admin",
		"Resource": "project [union-nav/development/dataplattform]\nproject [union-nav/production/dataplattform]\nproject [union-nav/staging/dataplattform]"
	}
]`,
			want: []Permission{
				{
					Name: "admin",
					Role: "admin",
					Resources: []Resource{
						{Kind: "organization", Organization: "union-nav"},
					},
				},
				{
					Name: "dataplattform-project-owners",
					Role: "admin",
					Resources: []Resource{
						{Kind: "project", Organization: "union-nav", Domain: "development", Project: "dataplattform"},
						{Kind: "project", Organization: "union-nav", Domain: "production", Project: "dataplattform"},
						{Kind: "project", Organization: "union-nav", Domain: "staging", Project: "dataplattform"},
					},
				},
			},
		},
		{
			name:  "pure JSON input without preamble",
			input: `[{"Name":"admin","Role":"admin","Resource":"organization [union-nav]"}]`,
			want: []Permission{
				{
					Name: "admin",
					Role: "admin",
					Resources: []Resource{
						{Kind: "organization", Organization: "union-nav"},
					},
				},
			},
		},
		{
			name:  "empty JSON array",
			input: `[]`,
			want:  []Permission{},
		},
		{
			name:    "no JSON in output",
			input:   "something went wrong, no json here",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `[{"Name": "broken"`,
			wantErr: true,
		},
		{
			name:    "resource field without brackets returns error",
			input:   `[{"Name":"admin","Role":"admin","Resource":"organization"}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParsePermissions([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParsePermissions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
