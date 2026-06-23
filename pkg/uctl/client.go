package uctl

import (
	"context"
	"log/slog"
)

type UCTLClient struct {
	config UnionConfig
}

func NewUCTLClient(config UnionConfig) UCTLClient {
	return UCTLClient{
		config,
	}
}

func (c *UCTLClient) GetIdentityAssignments(ctx context.Context, user string) ([]Permission, error) {
	cmd := NewUCTLCommand(c.config).Get().Identityassignments(user)
	slog.Debug(cmd.String())

	result, err := cmd.Exec(ctx)
	if err != nil {
		return nil, err
	}

	permissions, err := ParsePermissions([]byte(result))
	if err != nil {
		return nil, err
	}

	return permissions, nil
}
