package service

import (
	"fmt"

	redisfailoverv1 "github.com/spotahome/redis-operator/api/redisfailover/v1"
)

// GetRedisShutdownConfigMapName returns the name for redis configmap
func GetRedisShutdownConfigMapName(rf *redisfailoverv1.RedisFailover) string {
	if rf.Spec.Redis.ShutdownConfigMap != "" {
		return rf.Spec.Redis.ShutdownConfigMap
	}
	return GetRedisShutdownName(rf)
}

// GetRedisName returns the name for redis resources
func GetRedisName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(redisName, rf.Name)
}

// GetRedisShutdownName returns the name for redis resources
func GetRedisShutdownName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(redisShutdownName, rf.Name)
}

// GetRedisReadinessName returns the name for redis resources
func GetRedisReadinessName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(redisReadinessName, rf.Name)
}

// GetSentinelName returns the name for sentinel resources
func GetSentinelName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(sentinelName, rf.Name)
}

func GetRedisMasterName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(redisMasterName, rf.Name)
}

func GetRedisSlaveName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(redisSlaveName, rf.Name)
}

// GetTLSCertificateName returns the name for the cert-manager Certificate
// the operator creates when spec.tls.certManager is configured.
func GetTLSCertificateName(rf *redisfailoverv1.RedisFailover) string {
	return generateName(tlsCertificateName, rf.Name)
}

// GetTLSSecretName returns the Secret name that holds the TLS material
// for the cluster. It honors an explicit override when set; otherwise
// it derives from the failover name.
//
// Precedence:
//  1. spec.tls.certificateSecret.secretName (bring-your-own-secret mode)
//  2. spec.tls.certManager.secretName       (cert-manager override)
//  3. <generated default>                   (cert-manager managed)
//
// Returns the empty string when TLS is disabled or unconfigured.
func GetTLSSecretName(rf *redisfailoverv1.RedisFailover) string {
	tls := rf.Spec.TLS
	if tls == nil || !tls.Enabled {
		return ""
	}
	if tls.CertificateSecret != nil && tls.CertificateSecret.SecretName != "" {
		return tls.CertificateSecret.SecretName
	}
	if tls.CertManager != nil && tls.CertManager.SecretName != "" {
		return tls.CertManager.SecretName
	}
	return generateName(tlsCertificateName, rf.Name)
}

// TLSEnabled is a small helper used pervasively in the generator.
func TLSEnabled(rf *redisfailoverv1.RedisFailover) bool {
	return rf.Spec.TLS != nil && rf.Spec.TLS.Enabled
}

func generateName(typeName, metaName string) string {
	return fmt.Sprintf("%s%s-%s", baseName, typeName, metaName)
}
