package certs

import (
	"crypto/x509/pkix"
	"io/ioutil"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateCertificates(t *testing.T) {
	c, err := NewCACertificate(pkix.Name{CommonName: "test"}, int64(1), metav1.Duration{time.Hour * 24 * 60}, time.Now)
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Certificates) != 1 {
		t.Fatalf("expected 1 certificate in the chain, but got %d", len(c.Certificates))
	}

	validCerts := FilterExpiredCerts(c.Certificates...)
	if len(validCerts) != 1 {
		t.Fatalf("expected 1 valid certificate in the chain, but got %d", len(validCerts))
	}
}

func TestValidateCertificatesExpired(t *testing.T) {
	certBytes, err := ioutil.ReadFile("./testfiles/tls-expired.crt")
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	certs, err := ParseCertificateBytes(certBytes, nil)
	if err != nil {
		t.Fatal(err)
	}

	newCert, err := NewCACertificate(pkix.Name{CommonName: "etcdproxy-tests"}, int64(1), metav1.Duration{time.Hour * 24 * 60}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	certs.Certificates = append(certs.Certificates, newCert.Certificates...)

	if len(certs.Certificates) != 2 {
		t.Fatalf("expected 2 certificate in the chain, but got %d", len(certs.Certificates))
	}

	validCerts := FilterExpiredCerts(certs.Certificates...)
	if len(validCerts) != 1 {
		t.Fatalf("expected 1 valid certificate in the chain, but got %d", len(validCerts))
	}
}

func TestParseCertificateBytes(t *testing.T) {
	certBytes, err := ioutil.ReadFile("./testfiles/tls.crt")
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := ioutil.ReadFile("./testfiles/tls.key")
	if err != nil {
		t.Fatal(err)
	}

	certs, err := ParseCertificateBytes(certBytes, keyBytes)
	if err != nil {
		t.Fatal(err)
	}

	if len(certs.Certificates) != 1 {
		t.Fatalf("expected 1 certificate in bundle, but got %d", len(certs.Certificates))
	}
	if certs.Certificates[0].Issuer.CommonName != "etcdproxy-tests" {
		t.Fatalf("expected cn: 'etcdproxy-tests', but got '%s'", certs.Certificates[0].Issuer.CommonName)
	}
	if certs.Key == nil {
		t.Fatal("expected key, but key not found")
	}
}

func TestParseCertificateBytesCertOnly(t *testing.T) {
	certBytes, err := ioutil.ReadFile("./testfiles/tls.crt")
	if err != nil {
		t.Fatal(err)
	}

	certs, err := ParseCertificateBytes(certBytes, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(certs.Certificates) != 1 {
		t.Fatalf("expected 1 certificate in bundle, but got %d", len(certs.Certificates))
	}
	if certs.Certificates[0].Issuer.CommonName != "etcdproxy-tests" {
		t.Fatalf("expected cn: 'etcdproxy-tests', but got '%s'", certs.Certificates[0].Issuer.CommonName)
	}
	if certs.Key != nil {
		t.Fatal("did not expected key, but key found")
	}
}

func TestParseCertificateBytesMultipleCerts(t *testing.T) {
	certBytes, err := ioutil.ReadFile("./testfiles/tls-multiple.crt")
	if err != nil {
		t.Fatal(err)
	}

	certs, err := ParseCertificateBytes(certBytes, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(certs.Certificates) != 3 {
		t.Fatalf("expected 1 certificate in bundle, but got %d", len(certs.Certificates))
	}
	for _, c := range certs.Certificates {
		if c.Issuer.CommonName != "etcdproxy-tests" {
			t.Fatalf("expected cn: 'etcdproxy-tests', but got '%s'", certs.Certificates[0].Issuer.CommonName)
		}
	}
	if certs.Key != nil {
		t.Fatal("did not expected key, but key found")
	}
}
