package redis

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildTLSConfigNilOnEmptyCA(t *testing.T) {
	cfg, err := BuildTLSConfig(nil, "redis.example.com")
	assert.NoError(t, err)
	assert.Nil(t, cfg)

	cfg, err = BuildTLSConfig([]byte{}, "redis.example.com")
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestBuildTLSConfigRejectsEmptyServerName(t *testing.T) {
	ca := selfSignedCAPEM(t)
	_, err := BuildTLSConfig(ca, "")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "ServerName must not be empty")
	}
}

func TestBuildTLSConfigRejectsInvalidCA(t *testing.T) {
	_, err := BuildTLSConfig([]byte("not a pem"), "redis.example.com")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "no usable certificates")
	}
}

func TestBuildTLSConfigSucceeds(t *testing.T) {
	ca := selfSignedCAPEM(t)
	cfg, err := BuildTLSConfig(ca, "redis.example.com")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, cfg) {
		return
	}
	assert.Equal(t, "redis.example.com", cfg.ServerName)
	assert.NotNil(t, cfg.RootCAs)
	assert.Equal(t, uint16(0x0303), cfg.MinVersion, "MinVersion must be TLS 1.2 (0x0303)")
}

// selfSignedCAPEM produces a freshly-minted self-signed CA certificate
// in PEM form for use in BuildTLSConfig tests.
func selfSignedCAPEM(t *testing.T) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
