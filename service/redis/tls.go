package redis

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
)

// BuildTLSConfig assembles the *tls.Config the operator uses to dial
// Redis and Sentinel pods over TLS.
//
// The CA pool is loaded from the supplied caPEM, which the operator
// reads from the same Secret that cert-manager (or the user) populated
// for the cluster pods. The ServerName drives certificate validation:
// the operator dials pods by their (unstable) IP but presents a stable
// DNS name during the TLS handshake, which the certificate's DNS SANs
// cover.
//
// Returns (nil, nil) when caPEM is empty: callers treat a nil config as
// "no TLS" and dial in plaintext.
func BuildTLSConfig(caPEM []byte, serverName string) (*tls.Config, error) {
	if len(caPEM) == 0 {
		return nil, nil
	}
	if serverName == "" {
		return nil, errors.New("tls: ServerName must not be empty when caPEM is provided")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("tls: failed to parse CA certificate (no usable certificates found)")
	}
	return &tls.Config{
		RootCAs:    pool,
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}, nil
}
