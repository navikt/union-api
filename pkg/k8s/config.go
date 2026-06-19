package k8s

import "fmt"

// K8sConfig holds the configuration needed to connect to a GKE cluster
// via the Connect Gateway.
type K8sConfig struct {
	// FleetHostProjectNumber is the numeric GCP project number of the fleet
	// host project (not the cluster's project).
	FleetHostProjectNumber string
	// Location is the GCP region the fleet membership is registered in.
	// Example: "europe-west1"
	Location string
	// MembershipName is the fleet membership name the cluster was registered under.
	// Example: "dev-union-restricted"
	MembershipName string
}

// ConnectGatewayURL returns the full Connect Gateway base URL for the configured cluster.
func (c K8sConfig) ConnectGatewayURL() string {
	return c.host() + c.pathPrefix()
}

// host returns the regional Connect Gateway hostname for this config.
func (c K8sConfig) host() string {
	return fmt.Sprintf("https://%s-connectgateway.googleapis.com", c.Location)
}

// pathPrefix returns the Connect Gateway URL path for the configured cluster.
// This is prepended to every Kubernetes API request path by gatewayTransport.
func (c K8sConfig) pathPrefix() string {
	return fmt.Sprintf(
		"/v1/projects/%s/locations/%s/gkeMemberships/%s",
		c.FleetHostProjectNumber,
		c.Location,
		c.MembershipName,
	)
}
