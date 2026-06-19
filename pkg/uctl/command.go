package uctl

import (
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

func (c UCTLCommand) Exec() ([]byte, error) {
	cmd := exec.Command(c.command, c.args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute uctl command %s: %w", c.String(), err)
	}

	return output, nil
}

func (c UCTLCommand) String() string {
	return fmt.Sprintf("%s %s", c.command, strings.Join(c.args, " "))
}
