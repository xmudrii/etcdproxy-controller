package certs

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
)

var (
	// TODO: make this configurable.
	// TODO: set limits (e.g. min = 5 minutes)
	CABundleValidForDays     = 365 * 5 // 5 years.
	ServerBundleValidForDays = 365 * 3 // 3 years.
	ClientBundleValidForDays = 30      // 1 month.
)

// Certificate contains slice of certificates and a key.
type Certificate struct {
	Certificates []*x509.Certificate
	Key          crypto.PrivateKey
}

// newKeyPair generates new public and private key.
func newKeyPair() (crypto.PublicKey, crypto.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return &privateKey.PublicKey, privateKey, nil
}

// signCertificate signs provided certificate using issuer certificate and key.
func signCertificate(cert *x509.Certificate, certPublicKey crypto.PublicKey, issuerCertificate *x509.Certificate,
	issuerKey crypto.PrivateKey) (*x509.Certificate, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, cert, issuerCertificate, certPublicKey, issuerKey)
	if err != nil {
		return nil, err
	}

	certs, err := x509.ParseCertificates(derBytes)
	if err != nil {
		return nil, err
	}
	if len(certs) != 1 {
		return nil, fmt.Errorf("expected one certificate, but got %d", len(certs))
	}

	return certs[0], nil
}

// signCertificate is wrapper around basic signCertificate function, which takes issuer certificate from provided
// Certificate struct.
func (c *Certificate) signCertificate(cert *x509.Certificate, certPublicKey crypto.PublicKey) (*x509.Certificate, error) {
	return signCertificate(cert, certPublicKey, c.Certificates[0], c.Key)
}

func ipAddressesDNSNames(hosts []string) ([]net.IP, []string) {
	var ips []net.IP
	var dns []string

	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			ips = append(ips, ip)
		} else {
			dns = append(dns, host)
		}
	}

	// Include IP addresses as DNS subjectAltNames in the cert as well, for the sake of Python, Windows (< 10), and unnamed other libraries
	// Ensure these technically invalid DNS subjectAltNames occur after the valid ones, to avoid triggering cert errors in Firefox
	// See https://bugzilla.mozilla.org/show_bug.cgi?id=1148766
	for _, ip := range ips {
		dns = append(dns, ip.String())
	}

	return ips, dns
}

// EncodeCertificates converts x509 Certificate to bytes.
func EncodeCertificates(certs ...*x509.Certificate) ([]byte, error) {
	b := bytes.Buffer{}
	for _, cert := range certs {
		if err := pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return []byte{}, err
		}
	}
	return b.Bytes(), nil
}

// EncodeKey converts private key to bytes.
func EncodeKey(key crypto.PrivateKey) ([]byte, error) {
	b := bytes.Buffer{}
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		keyBytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return []byte{}, err
		}
		if err := pem.Encode(&b, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
			return b.Bytes(), err
		}
	case *rsa.PrivateKey:
		if err := pem.Encode(&b, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
			return []byte{}, err
		}
	default:
		return []byte{}, fmt.Errorf("unrecognized key type")

	}
	return b.Bytes(), nil
}
