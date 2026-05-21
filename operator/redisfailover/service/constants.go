package service

// variables refering to the redis exporter port
const (
	exporterPort                  = 9121
	sentinelExporterPort          = 9355
	exporterPortName              = "http-metrics"
	exporterContainerName         = "redis-exporter"
	sentinelExporterContainerName = "sentinel-exporter"
	exporterDefaultRequestCPU     = "10m"
	exporterDefaultLimitCPU       = "1000m"
	exporterDefaultRequestMemory  = "50Mi"
	exporterDefaultLimitMemory    = "100Mi"
)

const (
	baseName               = "rf"
	sentinelName           = "s"
	sentinelRoleName       = "sentinel"
	sentinelConfigFileName = "sentinel.conf"
	redisConfigFileName    = "redis.conf"
	redisName              = "r"
	redisMasterName        = "rm"
	redisSlaveName         = "rs"
	redisShutdownName      = "r-s"
	redisReadinessName     = "r-readiness"
	redisRoleName          = "redis"
	tlsCertificateName     = "tls"
	appLabel               = "redis-failover"
	hostnameTopologyKey    = "kubernetes.io/hostname"
)

// TLS volume / file layout.
//
// cert-manager produces a Secret with three well-known keys (tls.crt,
// tls.key, ca.crt). The operator mounts that Secret read-only into every
// Redis and Sentinel pod at tlsMountPath. The probe scripts, redis.conf
// and sentinel.conf reference the same paths.
const (
	tlsVolumeName  = "redis-tls"
	tlsMountPath   = "/tls"
	tlsCertFile    = tlsMountPath + "/tls.crt"
	tlsKeyFile     = tlsMountPath + "/tls.key"
	tlsCAFile      = tlsMountPath + "/ca.crt"
	tlsSecretKey   = "tls.crt"
	tlsSecretCAKey = "ca.crt"
)

const (
	redisRoleLabelKey    = "redisfailovers-role"
	redisRoleLabelMaster = "master"
	redisRoleLabelSlave  = "slave"
)
