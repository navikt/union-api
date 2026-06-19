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

type Resource struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}
