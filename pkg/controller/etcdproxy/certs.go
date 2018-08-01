package etcdproxy

import (
	"crypto/x509/pkix"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"github.com/xmudrii/etcdproxy-controller/pkg/certs"
)

// ProxyCertificateExpiryAnnotation is annotating are certificates successfully created and stored in the Secret or ConfigMap.
const ProxyCertificateExpiryAnnotation = "etcd.xmudrii.com/certificate-generated"

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
	serverSignerCert, _, err := serverSigner.GetPEMBytes()
	if err != nil {
		return err
	}

	// Generate server certificate/key pair.
	serverBundle, err := serverSigner.NewServerCertificate(pkix.Name{CommonName: serviceUrl},
		[]string{serviceUrl}, r.Int63n(100000), currentTime)
	if err != nil {
		return err
	}
	serverCertBytes, serverKeyBytes, err := serverBundle.GetPEMBytes()
	if err != nil {
		return err
	}

	// Write pairs to appropriate ConfigMaps and Secrets.
	var errs []error
	for _, configMap := range etcdstorage.Spec.CACertConfigMaps {
		err = c.ensureClientCABundle(etcdstorage, configMap, serverSignerCert, serverSigner.Certificates[0].NotAfter)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return utilerrors.NewAggregate(errs)
	}

	return c.ensureServerCert(etcdstorage, serverCertBytes, serverKeyBytes, serverBundle.Certificates[0].NotAfter)
}

// setNewAPIServerCertificates generates new self-signed client signer and client certificate/key pair.
// Self-signed client signer cert is stored in Controller ConfigMap to be used as CA, while client cert/key pair
// is stored in Controller Secret to be used by apiserver.
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
	clientSignerCert, _, err := clientSigner.GetPEMBytes() // goes to etcdproxy
	if err != nil {
		return err
	}

	// Write CA certificate to EtcdProxyController CA ConfigMap.
	err = c.ensureServingCABundle(etcdstorage, clientSignerCert, clientSigner.Certificates[0].NotAfter)

	// Generate client certificate for each Secret provided.
	var errs []error
	for _, secret := range etcdstorage.Spec.ClientCertSecrets {
		// Generate client certificate/key pair.
		clientBundle, err := clientSigner.NewClientCertificate(pkix.Name{CommonName: fmt.Sprintf("client-%s-%s",
			secret.Namespace, secret.Name)},
			r.Int63n(100000), currentTime)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		clientCertBytes, clientKeyBytes, err := clientBundle.GetPEMBytes() // goes to apiserver
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Write certificate and key pair to the Secret.
		err = c.ensureClientCert(etcdstorage, secret, clientCertBytes, clientKeyBytes, clientBundle.Certificates[0].NotAfter)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// ensureServingCABundle ensures the ConfigMap contains the required Serving CA bundle.
// The Serving CA bundle is located in the controller namespace and is used by the etcd-proxy to verify API server identity.
func (c *EtcdProxyController) ensureServingCABundle(etcdstorage *etcdstoragev1alpha1.EtcdStorage, caBytes []byte, expiryDate time.Time) error {
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdProxyCAConfigMapName(etcdstorage),
			Namespace: c.config.ControllerNamespace,
			// TODO: Remove this once validation is in place.
			Annotations: map[string]string{
				ProxyCertificateExpiryAnnotation: expiryDate.Format(time.RFC3339),
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Data: map[string]string{
			"serving-ca.crt": string(caBytes),
		},
	}

	return ensureConfigMap(c.kubeclientset, configMap)
}

// ensureClientCABundle ensures the ConfigMap contains the required Client CA bundle.
// The Client CA bundle is located in the API server namespace and is used by the API server to verify etcd-proxy identity.
func (c *EtcdProxyController) ensureClientCABundle(etcdstorage *etcdstoragev1alpha1.EtcdStorage, caDestination etcdstoragev1alpha1.CABundleDestination, caBytes []byte, expiryDate time.Time) error {
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caDestination.Name,
			Namespace: caDestination.Namespace,
			// TODO: Remove this once validation is in place.
			Annotations: map[string]string{
				ProxyCertificateExpiryAnnotation: expiryDate.Format(time.RFC3339),
			},
		},
		Data: map[string]string{
			"client-ca.crt": string(caBytes),
		},
	}

	return ensureConfigMap(c.kubeclientset, configMap)
}

// ensureServerCert ensures the Secret contains the required Server Certificate and Key.
// The server certificate is used by etcd-proxy.
func (c *EtcdProxyController) ensureServerCert(etcdstorage *etcdstoragev1alpha1.EtcdStorage, certBytes, keyBytes []byte, expiryDate time.Time) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdProxyServerCertsSecret(etcdstorage),
			Namespace: c.config.ControllerNamespace,
			Annotations: map[string]string{
				ProxyCertificateExpiryAnnotation: expiryDate.Format(time.RFC3339),
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certBytes,
			"tls.key": keyBytes,
		},
	}

	return ensureSecret(c.kubeclientset, secret)
}

// ensureClientCert ensures the Secret contains the required Client Certificate and Key.
// The client certificate is used by the API server to authenticate with etcd-proxy.
func (c *EtcdProxyController) ensureClientCert(etcdstorage *etcdstoragev1alpha1.EtcdStorage, certDestination etcdstoragev1alpha1.ClientCertificateDestination, certBytes, keyBytes []byte, expiryDate time.Time) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certDestination.Name,
			Namespace: certDestination.Namespace,
			Annotations: map[string]string{
				ProxyCertificateExpiryAnnotation: expiryDate.Format(time.RFC3339),
			},
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certBytes,
			"tls.key": keyBytes,
		},
	}

	return ensureSecret(c.kubeclientset, secret)
}
