package k8s

import "fmt"

// GKEConfig holds the configuration needed to connect to a GKE cluster
// via the Connect Gateway.
type GKEConfig struct {
	// FleetHostProjectNumber is the numeric GCP project number of the fleet
	// host project (not the cluster's project).
	FleetHostProjectNumber string `yaml:"fleet_host_project_number"`
	// Location is the GCP region the fleet membership is registered in.
	// Example: "europe-west1"
	Location string `yaml:"fleet_location"`
	// MembershipName is the fleet membership name the cluster was registered under.
	// Example: "dev-union-restricted"
	MembershipName string `yaml:"fleet_membership_name"`
}

// ConnectGatewayURL returns the full Connect Gateway base URL for the configured
// cluster, including the regional host and the membership path prefix.
//
// The path component is significant: client-go treats a path set on
// rest.Config.Host as a prefix applied to every request URL, which is exactly
// what the Connect Gateway requires. The https:// scheme must be present, or
// client-go rejects a host that carries a path.
func (c GKEConfig) ConnectGatewayURL() string {
	return fmt.Sprintf(
		"https://%s-connectgateway.googleapis.com/v1/projects/%s/locations/%s/gkeMemberships/%s",
		c.Location,
		c.FleetHostProjectNumber,
		c.Location,
		c.MembershipName,
	)
}
