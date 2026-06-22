package uctl

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParsePermissions(output []byte) ([]Permission, error) {
	jsonBytes, err := extractJSON(output)
	if err != nil {
		return nil, err
	}

	var raw []RawPermission
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	permissions := make([]Permission, 0, len(raw))

	for _, r := range raw {
		resource, err := parseResources(r.Resource)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, Permission{
			Name:      r.Name,
			Role:      r.Role,
			Resources: resource,
		})
	}

	return permissions, nil
}

func extractJSON(output []byte) ([]byte, error) {
	for i, b := range output {
		if b == '[' || b == '{' {
			return output[i:], nil
		}
	}

	return nil, fmt.Errorf("no json object or array found in output")
}

func parseResources(input string) ([]Resource, error) {
	lines := strings.Split(input, "\n")

	resources := make([]Resource, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		resource, err := parseResource(line)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func parseResource(line string) (Resource, error) {
	open := strings.Index(line, "[")
	close := strings.LastIndex(line, "]")

	if open == -1 || close == -1 || close <= open {
		return Resource{}, fmt.Errorf("unable to parse resource from line: %q", line)
	}

	kind, err := parseKind(strings.TrimSpace(line[:open]))
	if err != nil {
		return Resource{}, err
	}
	path := strings.Split(strings.TrimSpace(line[open+1:close]), "/")

	return Resource{
		Kind:         kind,
		Organization: pathSegment(path, 0),
		Domain:       pathSegment(path, 1),
		Project:      pathSegment(path, 2),
	}, nil
}

func parseKind(s string) (Kind, error) {
	switch Kind(s) {
	case Organization, Project, Domain:
		return Kind(s), nil
	default:
		return "", fmt.Errorf("unknown kind: %q", s)
	}
}

func pathSegment(parts []string, i int) string {
	if i < len(parts) {
		return parts[i]
	}
	return ""
}
