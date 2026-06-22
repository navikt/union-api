package serviceaccounts

type ServiceAccount struct {
	K8sServiceAccount    string `json:"k8sServiceAccount"`
	GoogleServiceAccount string `json:"googleServiceAccount"`
	UnionProject         string `json:"unionProject"`
	UnionDomain          string `json:"unionDomain"`
}
