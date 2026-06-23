package uctl

import (
	"context"
	"testing"
)

var testConfig = UnionConfig{
	Org:                "test-org",
	ClientID:           "test-client-id",
	ClientSecretEnvVar: "TEST_SECRET",
	Endpoint:           "https://test.endpoint",
}

const baseArgs = "uctl --org test-org --admin.authType ClientSecret --admin.clientId test-client-id --admin.clientSecretEnvVar TEST_SECRET --admin.endpoint https://test.endpoint --output json"

func TestUCTLCommand_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build func() UCTLCommand
		want  string
	}{
		{
			name:  "base command from config",
			build: func() UCTLCommand { return NewUCTLCommand(testConfig) },
			want:  baseArgs,
		},
		{
			name:  "get subcommand",
			build: func() UCTLCommand { return NewUCTLCommand(testConfig).Get() },
			want:  baseArgs + " get",
		},
		{
			name:  "user-info subcommand",
			build: func() UCTLCommand { return NewUCTLCommand(testConfig).UserInfo() },
			want:  baseArgs + " user-info",
		},
		{
			name:  "get identityassignments for user",
			build: func() UCTLCommand { return NewUCTLCommand(testConfig).Get().Identityassignments("alice") },
			want:  baseArgs + " get identityassignments --user alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.build().String(); got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestUCTLCommand_Exec(t *testing.T) {
	t.Parallel()

	t.Run("returns stdout on success", func(t *testing.T) {
		t.Parallel()

		cmd := UCTLCommand{command: "echo", args: []string{"hello"}}
		out, err := cmd.Exec(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// echo appends a newline
		if string(out) != "hello\n" {
			t.Errorf("got %q, want %q", string(out), "hello\n")
		}
	})

	t.Run("returns wrapped error on failure", func(t *testing.T) {
		t.Parallel()

		cmd := UCTLCommand{command: "false"}
		_, err := cmd.Exec(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
