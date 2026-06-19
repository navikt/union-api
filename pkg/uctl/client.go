package uctl

import (
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

func (c *UCTLClient) GetIdentityAssignments(user string) ([]Permission, error) {
	result, err := NewUCTLCommand(c.config).Get().Identityassignments(user).Exec()

	slog.Debug(NewUCTLCommand(c.config).Get().Identityassignments(user).String())

	if err != nil {
		return nil, err
	}

	permissions, err := ParsePermissions([]byte(result))
	if err != nil {
		return nil, err
	}

	return permissions, nil
}
