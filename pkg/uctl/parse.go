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
		permissions = append(permissions, Permission{
			Name:      r.Name,
			Role:      r.Role,
			Resources: parseResources(r.Resource),
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

func parseResources(input string) []Resource {
	lines := strings.Split(input, "\n")

	resources := make([]Resource, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		resource, ok := parseResource(line)
		if ok {
			resources = append(resources, resource)
		}
	}

	return resources
}

func parseResource(line string) (Resource, bool) {
	open := strings.Index(line, "[")
	close := strings.LastIndex(line, "]")

	if open == -1 || close == -1 || close <= open {
		return Resource{}, false
	}

	kind := strings.TrimSpace(line[:open])
	path := strings.TrimSpace(line[open+1 : close])

	return Resource{
		Kind: kind,
		Path: path,
	}, true
}
