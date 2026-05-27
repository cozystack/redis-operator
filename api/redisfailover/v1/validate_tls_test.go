package v1

import (
	"testing"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
)

func TestValidateTLS(t *testing.T) {
	tests := []struct {
		name              string
		tls               *TLSSettings
		expectedError     string
		expectedAuth      string
		expectedIssuerKnd string
		expectedIssuerGrp string
	}{
		{
			name: "nil TLS is allowed",
			tls:  nil,
		},
		{
			name: "disabled TLS skips validation even with garbage",
			tls: &TLSSettings{
				Enabled:     false,
				AuthClients: "garbage",
			},
		},
		{
			name: "enabled without provider rejected",
			tls: &TLSSettings{
				Enabled: true,
			},
			expectedError: "neither tls.certManager nor tls.certificateSecret is set",
		},
		{
			name: "both providers rejected",
			tls: &TLSSettings{
				Enabled: true,
				CertManager: &CertManagerSettings{
					IssuerRef: cmmeta.IssuerReference{Name: "ca"},
				},
				CertificateSecret: &LocalSecretReference{SecretName: "tls"},
			},
			expectedError: "mutually exclusive",
		},
		{
			name: "certManager without issuerRef name rejected",
			tls: &TLSSettings{
				Enabled:     true,
				CertManager: &CertManagerSettings{},
			},
			expectedError: "tls.certManager.issuerRef.name is required",
		},
		{
			name: "certManager defaults kind and group",
			tls: &TLSSettings{
				Enabled: true,
				CertManager: &CertManagerSettings{
					IssuerRef: cmmeta.IssuerReference{Name: "my-ca"},
				},
			},
			expectedAuth:      TLSAuthClientsNo,
			expectedIssuerKnd: "Issuer",
			expectedIssuerGrp: "cert-manager.io",
		},
		{
			name: "certManager preserves explicit kind and group",
			tls: &TLSSettings{
				Enabled: true,
				CertManager: &CertManagerSettings{
					IssuerRef: cmmeta.IssuerReference{
						Name:  "my-ca",
						Kind:  "ClusterIssuer",
						Group: "example.com",
					},
				},
			},
			expectedAuth:      TLSAuthClientsNo,
			expectedIssuerKnd: "ClusterIssuer",
			expectedIssuerGrp: "example.com",
		},
		{
			name: "certificateSecret without secretName rejected",
			tls: &TLSSettings{
				Enabled:           true,
				CertificateSecret: &LocalSecretReference{},
			},
			expectedError: "tls.certificateSecret.secretName is required",
		},
		{
			name: "invalid authClients rejected",
			tls: &TLSSettings{
				Enabled:           true,
				AuthClients:       "maybe",
				CertificateSecret: &LocalSecretReference{SecretName: "tls"},
			},
			expectedError: "tls.authClients must be one of",
		},
		{
			name: "authClients yes is accepted",
			tls: &TLSSettings{
				Enabled:           true,
				AuthClients:       TLSAuthClientsYes,
				CertificateSecret: &LocalSecretReference{SecretName: "tls"},
			},
			expectedAuth: TLSAuthClientsYes,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			rf := generateRedisFailover("test-tls", nil)
			rf.Spec.TLS = test.tls

			err := rf.Validate()

			if test.expectedError != "" {
				if a.Error(err) {
					a.Contains(err.Error(), test.expectedError)
				}
				return
			}

			if !a.NoError(err) {
				return
			}
			if test.tls == nil || !test.tls.Enabled {
				return
			}
			a.Equal(test.expectedAuth, rf.Spec.TLS.AuthClients)
			if test.tls.CertManager != nil {
				a.Equal(test.expectedIssuerKnd, rf.Spec.TLS.CertManager.IssuerRef.Kind)
				a.Equal(test.expectedIssuerGrp, rf.Spec.TLS.CertManager.IssuerRef.Group)
			}
		})
	}
}
