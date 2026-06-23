package uctl

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type UCTLCommand struct {
	command string
	args    []string
}

func NewUCTLCommand(cfg UnionConfig) UCTLCommand {
	return UCTLCommand{
		command: "uctl",
		args: []string{
			"--org", cfg.Org,
			"--admin.authType", "ClientSecret",
			"--admin.clientId", cfg.ClientID,
			"--admin.clientSecretEnvVar", cfg.ClientSecretEnvVar,
			"--admin.endpoint", cfg.Endpoint,
			"--output", "json",
		},
	}
}

func (c UCTLCommand) Get() UCTLCommand {
	c.args = append(c.args, "get")
	return c
}

func (c UCTLCommand) UserInfo() UCTLCommand {
	c.args = append(c.args, "user-info")
	return c
}

func (c UCTLCommand) Identityassignments(user string) UCTLCommand {
	c.args = append(c.args, "identityassignments", "--user", user)
	return c
}

func (c UCTLCommand) Exec(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.command, c.args...)
	output, err := cmd.Output()
	if err != nil {
		// Output() populates ExitError.Stderr when Stderr is unset; surface it
		// so failures from uctl are diagnosable instead of an opaque exit code.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("failed to execute uctl command %s: %w: %s", c.String(), err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to execute uctl command %s: %w", c.String(), err)
	}

	return output, nil
}

func (c UCTLCommand) String() string {
	return fmt.Sprintf("%s %s", c.command, strings.Join(c.args, " "))
}
