package uctl

type RawPermission struct {
	Name     string `json:"Name"`
	Role     string `json:"Role"`
	Resource string `json:"Resource"`
}

type Permission struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Resources []Resource `json:"resources"`
}

type Kind string

const (
	Organization Kind = "organization"
	Project      Kind = "project"
	Domain       Kind = "domain"
)

type Resource struct {
	Kind         Kind   `json:"kind"`
	Organization string `json:"organization"`
	Domain       string `json:"domain"`
	Project      string `json:"project"`
}
