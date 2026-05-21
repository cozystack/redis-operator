package service_test

import (
	"strings"
	"testing"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	redisfailoverv1 "github.com/spotahome/redis-operator/api/redisfailover/v1"
	"github.com/spotahome/redis-operator/log"
	"github.com/spotahome/redis-operator/metrics"
	mK8SService "github.com/spotahome/redis-operator/mocks/service/k8s"
	rfservice "github.com/spotahome/redis-operator/operator/redisfailover/service"
)

// generateTLSRF returns a RedisFailover with TLS enabled via cert-manager.
func generateTLSRF() *redisfailoverv1.RedisFailover {
	rf := generateRF()
	rf.Spec.Redis.Image = "redis:7.2"
	rf.Spec.Sentinel.Image = "redis:7.2"
	rf.Spec.Redis.Port = 6379
	rf.Spec.Redis.Exporter.Enabled = true
	rf.Spec.Sentinel.Exporter.Enabled = true
	rf.Spec.TLS = &redisfailoverv1.TLSSettings{
		Enabled:     true,
		AuthClients: redisfailoverv1.TLSAuthClientsNo,
		CertManager: &redisfailoverv1.CertManagerSettings{
			IssuerRef: cmmeta.IssuerReference{
				Name:  "test-ca",
				Kind:  "Issuer",
				Group: "cert-manager.io",
			},
		},
	}
	return rf
}

func ensureSucceeded(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("ensure call failed: %v", err)
	}
}

func TestRedisConfigMapHasTLSDirectives(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *corev1.ConfigMap
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateConfigMap", rf.Namespace, mock.MatchedBy(func(cm *corev1.ConfigMap) bool {
		if cm.Name == rfservice.GetRedisName(rf) {
			got = cm
			return true
		}
		return false
	})).Return(nil)

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureRedisConfigMap(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	conf := got.Data["redis.conf"]
	a.Contains(conf, "tls-port 6379")
	a.Contains(conf, "port 0")
	a.Contains(conf, "tls-cert-file /tls/tls.crt")
	a.Contains(conf, "tls-key-file /tls/tls.key")
	a.Contains(conf, "tls-ca-cert-file /tls/ca.crt")
	a.Contains(conf, "tls-auth-clients no")
	a.Contains(conf, "tls-replication yes")
	// Plain "port 6379" must not appear when TLS is on.
	a.NotContains(conf, "\nport 6379")
}

func TestSentinelConfigMapHasTLSDirectives(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *corev1.ConfigMap
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateConfigMap", rf.Namespace, mock.MatchedBy(func(cm *corev1.ConfigMap) bool {
		if cm.Name == rfservice.GetSentinelName(rf) {
			got = cm
			return true
		}
		return false
	})).Return(nil)

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureSentinelConfigMap(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	conf := got.Data["sentinel.conf"]
	a.Contains(conf, "tls-port 26379")
	a.Contains(conf, "port 0")
	a.Contains(conf, "tls-cert-file /tls/tls.crt")
	a.Contains(conf, "tls-auth-clients no")
	a.Contains(conf, "tls-replication yes")
}

func TestRedisStatefulSetHasTLSVolume(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *appsv1.StatefulSet
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateStatefulSet", rf.Namespace, mock.MatchedBy(func(ss *appsv1.StatefulSet) bool {
		got = ss
		return true
	})).Return(nil)
	ms.On("CreateOrUpdatePodDisruptionBudget", rf.Namespace, mock.Anything).Return(nil).Maybe()

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureRedisStatefulset(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	expectedSecret := rfservice.GetTLSSecretName(rf)

	// Volume present and points at the right secret.
	var tlsVol *corev1.Volume
	for i := range got.Spec.Template.Spec.Volumes {
		v := &got.Spec.Template.Spec.Volumes[i]
		if v.Name == "redis-tls" {
			tlsVol = v
			break
		}
	}
	if a.NotNil(tlsVol, "redis-tls volume must be present") {
		a.NotNil(tlsVol.Secret)
		if tlsVol.Secret != nil {
			a.Equal(expectedSecret, tlsVol.Secret.SecretName)
		}
	}

	// Mount present on the redis container.
	redisContainer := got.Spec.Template.Spec.Containers[0]
	var found bool
	for _, m := range redisContainer.VolumeMounts {
		if m.Name == "redis-tls" && m.MountPath == "/tls" && m.ReadOnly {
			found = true
			break
		}
	}
	a.True(found, "redis container must mount the TLS secret read-only at /tls")
}

func TestRedisExporterHasTLSEnv(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *appsv1.StatefulSet
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateStatefulSet", rf.Namespace, mock.MatchedBy(func(ss *appsv1.StatefulSet) bool {
		got = ss
		return true
	})).Return(nil)
	ms.On("CreateOrUpdatePodDisruptionBudget", rf.Namespace, mock.Anything).Return(nil).Maybe()

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureRedisStatefulset(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	var exporter *corev1.Container
	for i := range got.Spec.Template.Spec.Containers {
		c := &got.Spec.Template.Spec.Containers[i]
		if c.Name == "redis-exporter" {
			exporter = c
			break
		}
	}
	if !a.NotNil(exporter, "redis exporter sidecar must be present") {
		return
	}
	env := envToMap(exporter.Env)
	a.Equal("/tls/tls.crt", env["REDIS_EXPORTER_TLS_CLIENT_CERT_FILE"])
	a.Equal("/tls/tls.key", env["REDIS_EXPORTER_TLS_CLIENT_KEY_FILE"])
	a.Equal("/tls/ca.crt", env["REDIS_EXPORTER_TLS_CA_CERT_FILE"])
	a.Contains(env["REDIS_ADDR"], "rediss://")
}

func TestSentinelExporterTLSAddr(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *appsv1.Deployment
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateDeployment", rf.Namespace, mock.MatchedBy(func(d *appsv1.Deployment) bool {
		got = d
		return true
	})).Return(nil)
	ms.On("CreateOrUpdatePodDisruptionBudget", rf.Namespace, mock.Anything).Return(nil).Maybe()

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureSentinelDeployment(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	var exporter *corev1.Container
	for i := range got.Spec.Template.Spec.Containers {
		c := &got.Spec.Template.Spec.Containers[i]
		if c.Name == "sentinel-exporter" {
			exporter = c
			break
		}
	}
	if !a.NotNil(exporter, "sentinel exporter sidecar must be present") {
		return
	}
	env := envToMap(exporter.Env)
	a.Equal("rediss://127.0.0.1:26379", env["REDIS_ADDR"])
	a.Equal("/tls/tls.crt", env["REDIS_EXPORTER_TLS_CLIENT_CERT_FILE"])
}

func TestRedisShutdownScriptUsesTLS(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *corev1.ConfigMap
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateConfigMap", rf.Namespace, mock.MatchedBy(func(cm *corev1.ConfigMap) bool {
		if cm.Name == rfservice.GetRedisShutdownConfigMapName(rf) {
			got = cm
			return true
		}
		return false
	})).Return(nil)

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureRedisShutdownConfigMap(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	script := got.Data["shutdown.sh"]
	a.Contains(script, "--tls")
	a.Contains(script, "--cacert /tls/ca.crt")
	a.Contains(script, "--cert /tls/tls.crt")
	a.Contains(script, "--key /tls/tls.key")
}

func TestRedisReadinessScriptUsesTLS(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *corev1.ConfigMap
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateConfigMap", rf.Namespace, mock.MatchedBy(func(cm *corev1.ConfigMap) bool {
		if cm.Name == rfservice.GetRedisReadinessName(rf) {
			got = cm
			return true
		}
		return false
	})).Return(nil)

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureRedisReadinessConfigMap(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	script := got.Data["ready.sh"]
	a.Contains(script, "--tls")
	a.Contains(script, "--cacert /tls/ca.crt")
}

func TestRedisLivenessProbeUsesTLS(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *appsv1.StatefulSet
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateStatefulSet", rf.Namespace, mock.MatchedBy(func(ss *appsv1.StatefulSet) bool {
		got = ss
		return true
	})).Return(nil)
	ms.On("CreateOrUpdatePodDisruptionBudget", rf.Namespace, mock.Anything).Return(nil).Maybe()

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureRedisStatefulset(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	probe := got.Spec.Template.Spec.Containers[0].LivenessProbe
	if !a.NotNil(probe) || !a.NotNil(probe.Exec) {
		return
	}
	cmd := strings.Join(probe.Exec.Command, " ")
	a.Contains(cmd, "--tls")
	a.Contains(cmd, "--cacert /tls/ca.crt")
}

func TestSentinelProbeUsesTLS(t *testing.T) {
	a := assert.New(t)
	rf := generateTLSRF()

	var got *appsv1.Deployment
	ms := &mK8SService.Services{}
	ms.On("CreateOrUpdateDeployment", rf.Namespace, mock.MatchedBy(func(d *appsv1.Deployment) bool {
		got = d
		return true
	})).Return(nil)
	ms.On("CreateOrUpdatePodDisruptionBudget", rf.Namespace, mock.Anything).Return(nil).Maybe()

	gen := rfservice.NewRedisFailoverKubeClient(ms, log.DummyLogger{}, metrics.Dummy)
	ensureSucceeded(t, gen.EnsureSentinelDeployment(rf, nil, nil))

	if !a.NotNil(got) {
		return
	}
	probe := got.Spec.Template.Spec.Containers[0].LivenessProbe
	if !a.NotNil(probe) || !a.NotNil(probe.Exec) {
		return
	}
	cmd := strings.Join(probe.Exec.Command, " ")
	a.Contains(cmd, "--tls")
}

func TestTLSSecretNamePrecedence(t *testing.T) {
	a := assert.New(t)

	rfCM := generateTLSRF()
	a.Equal("rftls-"+rfCM.Name, rfservice.GetTLSSecretName(rfCM),
		"cert-manager managed mode should use generated default")

	rfCMOverride := generateTLSRF()
	rfCMOverride.Spec.TLS.CertManager.SecretName = "custom-secret"
	a.Equal("custom-secret", rfservice.GetTLSSecretName(rfCMOverride),
		"certManager.secretName should override the default")

	rfBYO := generateTLSRF()
	rfBYO.Spec.TLS.CertManager = nil
	rfBYO.Spec.TLS.CertificateSecret = &redisfailoverv1.LocalSecretReference{SecretName: "byo-tls"}
	a.Equal("byo-tls", rfservice.GetTLSSecretName(rfBYO),
		"certificateSecret.secretName should take precedence")

	rfOff := generateRF()
	a.Equal("", rfservice.GetTLSSecretName(rfOff),
		"with TLS disabled the helper must return an empty string")
}

func envToMap(env []corev1.EnvVar) map[string]string {
	out := make(map[string]string, len(env))
	for _, e := range env {
		out[e.Name] = e.Value
	}
	return out
}

