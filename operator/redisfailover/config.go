package redisfailover

// DefaultClusterDomain is the SAN suffix used when --cluster-domain is
// not supplied. cluster.local is the upstream Kubernetes default and
// keeps the operator's behaviour backwards-compatible.
const DefaultClusterDomain = "cluster.local"

// Config is the configuration for the redis operator.
type Config struct {
	ListenAddress            string
	MetricsPath              string
	Concurrency              int
	SupportedNamespacesRegex string
	// ClusterDomain is the cluster's DNS suffix (e.g. "cluster.local",
	// "cozy.local"). It is used to template the `*.svc.<domain>` SAN
	// entries on the cert-manager Certificate so that TLS verification
	// succeeds on clusters running a non-default --cluster-domain.
	// Empty falls back to DefaultClusterDomain.
	ClusterDomain string
}
