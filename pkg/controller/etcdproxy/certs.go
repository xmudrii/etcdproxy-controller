package etcdproxy

import (
	"crypto/x509/pkix"
	"fmt"
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"github.com/xmudrii/etcdproxy-controller/pkg/certs"
)

// annCertificateGenerated is annotating are certificates successfully created and stored in the Secret or ConfigMap.
const annCertificateGenerated = "etcd.xmudrii.com/certificate-generated"

// setNewEtcdProxyCertificates generates new self-signed server signer and server certificate/key pair.
// Self-signed server signer cert is stored in APIServer ConfigMap to be used as CA by etcd, while server cert/key pair
// is stored in Controller Secret to be used by etcdproxy.
func (c *EtcdProxyController) setNewEtcdProxyCertificates(etcdstorage *etcdstoragev1alpha1.EtcdStorage) error {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))
	serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(etcdstorage), c.config.ControllerNamespace)

	// TODO: implement certificates regeneration. this could require architecture change, because we could require
	// new secret for storing singer key.
	serverSigner, err := certs.NewCACertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-server-signer-%v", serviceUrl, time.Now().Unix()),
	}, r.Int63n(100000), currentTime)
	if err != nil {
		return err
	}
	serverSignerCert, err := certs.EncodeCertificates(serverSigner.Certificates...)
	if err != nil {
		return err
	}

	// Generate server certificate/key pair.
	serverBundle, err := serverSigner.NewServerCertificate(pkix.Name{CommonName: serviceUrl},
		[]string{serviceUrl}, r.Int63n(100000), currentTime)
	if err != nil {
		return err
	}
	serverCertBytes, err := certs.EncodeCertificates(serverBundle.Certificates...)
	if err != nil {
		return err
	}
	serverKeyBytes, err := certs.EncodeKey(serverBundle.Key)
	if err != nil {
		return err
	}

	// Write pairs to appropriate ConfigMaps and Secrets.
	err = c.updateAPIServerServingCAConfigMaps(etcdstorage, serverSignerCert)
	if err != nil {
		return err
	}
	return c.createEtcdProxyServingCertSecret(etcdstorage, serverCertBytes, serverKeyBytes)
}

// setNewAPIServerCertificates generates new self-signed client signer and client certificate/key pair.
// Self-signed client signer cert is stored in Controller ConfigMap to be used as CA, while client cert/key pair
// is stored in Controller Secret to be used by apiserver.
// TODO: this should generate new certificate for each configmap. But there's a problem here, as we need to match
// etcd ca certificate and client certificates.
func (c *EtcdProxyController) setNewAPIServerCertificates(etcdstorage *etcdstoragev1alpha1.EtcdStorage) error {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))
	serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(etcdstorage), c.config.ControllerNamespace)

	// TODO: implement certificates regeneration. this could require architecture change, because we could require
	// new secret for storing singer key.
	clientSigner, err := certs.NewCACertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-client-signer-%v", serviceUrl, time.Now().Unix()),
	}, r.Int63n(100000), currentTime)
	if err != nil {
		return err
	}
	clientSignerCert, err := certs.EncodeCertificates(clientSigner.Certificates...) // goes to etcdproxy
	if err != nil {
		return err
	}

	// Generate client certificate/key pair.
	clientBundle, err := clientSigner.NewClientCertificate(pkix.Name{CommonName: "client"},
		r.Int63n(100000), currentTime)
	if err != nil {
		return err
	}
	clientCertBytes, err := certs.EncodeCertificates(clientBundle.Certificates...) // goes to apiserver
	if err != nil {
		return err
	}
	clientKeyBytes, err := certs.EncodeKey(clientBundle.Key) // goes to apiserver
	if err != nil {
		return err
	}

	// Write pairs to appropriate ConfigMaps and Secrets.
	err = c.createEtcdProxyClientCAConfigMap(etcdstorage, clientSignerCert)
	if err != nil {
		return err
	}
	return c.updateAPIServerClientCertSecrets(etcdstorage, clientCertBytes, clientKeyBytes)
}
