package certs

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCACertificate generates and signs new CA certificate and key.
func NewCACertificate(subject pkix.Name, serialNumber int64, validity metav1.Duration, currentTime func() time.Time) (*Certificate, error) {
	caPublicKey, caPrivateKey, err := newKeyPair()
	if err != nil {
		return nil, err
	}

	caCert := &x509.Certificate{
		Subject: subject,

		SignatureAlgorithm: x509.SHA256WithRSA,

		NotBefore:    currentTime().Add(-1 * time.Second),
		NotAfter:     currentTime().Add(validity.Duration),
		SerialNumber: big.NewInt(serialNumber),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	cert, err := signCertificate(caCert, caPublicKey, caCert, caPrivateKey)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		Certificates: []*x509.Certificate{cert},
		Key:          caPrivateKey,
	}, nil
}

// NewServerCertificate generates and signs new Server certificate and key from CA bundle.
func (c *Certificate) NewServerCertificate(subject pkix.Name, hosts []string, serialNumber int64, validity metav1.Duration, currentTime func() time.Time) (*Certificate, error) {
	serverPublicKey, serverPrivateKey, err := newKeyPair()
	if err != nil {
		return nil, err
	}

	serverCert := &x509.Certificate{
		Subject: subject,

		SignatureAlgorithm: x509.SHA256WithRSA,

		NotBefore:    currentTime().Add(-1 * time.Second),
		NotAfter:     currentTime().Add(validity.Duration),
		SerialNumber: big.NewInt(serialNumber),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		// etcd requires from server key to be able to auth both server and client.
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	serverCert.IPAddresses, serverCert.DNSNames = ipAddressesDNSNames(hosts)

	cert, err := c.signCertificate(serverCert, serverPublicKey)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		Certificates: append([]*x509.Certificate{cert}, c.Certificates...),
		Key:          serverPrivateKey,
	}, nil
}

// NewClientCertificate generates and signs new Client certificate and key from server certificate..
func (c *Certificate) NewClientCertificate(subject pkix.Name, serialNumber int64, validity metav1.Duration, currentTime func() time.Time) (*Certificate, error) {
	clientPublicKey, clientPrivateKey, err := newKeyPair()
	if err != nil {
		return nil, err
	}

	clientCert := &x509.Certificate{
		Subject: subject,

		SignatureAlgorithm: x509.SHA256WithRSA,

		NotBefore:    currentTime().Add(-1 * time.Second),
		NotAfter:     currentTime().Add(validity.Duration),
		SerialNumber: big.NewInt(serialNumber),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	cert, err := c.signCertificate(clientCert, clientPublicKey)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		Certificates: []*x509.Certificate{cert},
		Key:          clientPrivateKey,
	}, nil
}
