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

	// Write CA certificate to EtcdProxyController CA ConfigMap.
	err = c.createEtcdProxyClientCAConfigMap(etcdstorage, clientSignerCert)
	if err != nil {
		return err
	}

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
		clientCertBytes, err := certs.EncodeCertificates(clientBundle.Certificates...) // goes to apiserver
		if err != nil {
			errs = append(errs, err)
			continue
		}
		clientKeyBytes, err := certs.EncodeKey(clientBundle.Key) // goes to apiserver
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Write certificate and key pair to the Secret.
		err = c.updateAPIServerClientCertSecrets(etcdstorage, secret, clientCertBytes, clientKeyBytes)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// createEtcdProxyClientCAConfigMap creates ConfigMap in controller namespace with Etcd Proxy CA bundle
// for verifying incoming client certificates.
func (c EtcdProxyController) createEtcdProxyClientCAConfigMap(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	clientSingerCert []byte) error {
	// ConfigMap in controller namespace for the etcd proxy CA certificate.
	_, err := c.kubeclientset.CoreV1().ConfigMaps(c.config.ControllerNamespace).
		Get(etcdProxyCAConfigMapName(etcdstorage), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		data := map[string]string{"client-ca.crt": string(clientSingerCert)}
		_, err = c.kubeclientset.CoreV1().ConfigMaps(c.config.ControllerNamespace).
			Create(newConfigMap(etcdstorage, etcdProxyCAConfigMapName(etcdstorage), c.config.ControllerNamespace, data))
		if err != nil {
			// TODO: refactor event handling (hint: see #40).
			c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
			return err
		}
		return nil
	}
	if err != nil {
		// TODO: refactor event handling (hint: see #40).
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
		return err
	}
	return nil
}

// createEtcdProxyServingCertSecret creates Secret in controller namespace with Etcd Proxy serving certificate and key.
func (c EtcdProxyController) createEtcdProxyServingCertSecret(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	serverCert, serverKey []byte) error {
	// Secret for the etcd proxy server certs in controller namespace.
	_, err := c.kubeclientset.CoreV1().Secrets(c.config.ControllerNamespace).
		Get(etcdProxyServerCertsSecret(etcdstorage), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		data := map[string][]byte{
			"tls.crt": serverCert,
			"tls.key": serverKey,
		}
		_, err = c.kubeclientset.CoreV1().Secrets(c.config.ControllerNamespace).
			Create(newSecret(etcdstorage, etcdProxyServerCertsSecret(etcdstorage), c.config.ControllerNamespace, data))
		if err != nil {
			// TODO: refactor event handling (hint: see #40).
			c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
			return err
		}
		return nil
	}
	if err != nil {
		// TODO: refactor event handling (hint: see #40).
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
		return err
	}
	return nil
}

// updateAPIServerServingCAConfigMaps updates the ConfigMap in the aggregated API server namespace with the CA certificate.
func (c *EtcdProxyController) updateAPIServerServingCAConfigMaps(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	serverSignerCert []byte) error {
	var errs []error
	// Check are ConfigMap name and namespace provided.
	for _, configMap := range etcdstorage.Spec.CACertConfigMaps {
		caConfigMap, err := c.kubeclientset.CoreV1().ConfigMaps(configMap.Namespace).
			Get(configMap.Name, metav1.GetOptions{})
		if err != nil {
			// TODO: refactor event handling (hint: see #40).
			c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
			errs = append(errs, err)
			continue
		}

		caConfigMapCopy := caConfigMap.DeepCopy()
		if val, ok := caConfigMapCopy.Annotations[annCertificateGenerated]; ok {
			if val == "true" {
				continue
			}
		}

		caConfigMapCopy.Data = map[string]string{"server-ca.crt": string(serverSignerCert)}

		// TODO: extend annotations to include more information about certs, including expiry date, etc.
		// HINT: take a look at openshift/service-serving-cert-signer for ideas.
		caConfigMapCopy.Annotations = map[string]string{
			annCertificateGenerated: "true",
		}

		// Check are ConfigMaps different and perform update if they are.
		if !equality.Semantic.DeepEqual(caConfigMap, caConfigMapCopy) {
			_, err = c.kubeclientset.CoreV1().ConfigMaps(caConfigMapCopy.Namespace).Update(caConfigMapCopy)
			if err != nil {
				// TODO: refactor event handling (hint: see #40).
				c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
				errs = append(errs, err)
			}
		}
	}

	return utilerrors.NewAggregate(errs)
}

// updateAPIServerClientCertSecrets updates the Secret in the aggregated API server namespace
// with the client certificate and key.
func (c *EtcdProxyController) updateAPIServerClientCertSecrets(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	secret etcdstoragev1alpha1.ClientCertificateDestination, clientCert, clientKey []byte) error {
	certSecret, err := c.kubeclientset.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	if err != nil {
		// TODO: refactor event handling (hint: see #40).
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
		return err
	}

	certSecretCopy := certSecret.DeepCopy()

	if certSecretCopy.Type != corev1.SecretTypeTLS {
		err = fmt.Errorf("certificates secret must be type of kubernetes.io/tls")
		// TODO: refactor event handling (hint: see #40).
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
		return err
	}
	if val, ok := certSecretCopy.Annotations[annCertificateGenerated]; ok {
		if val == "true" {
			return nil
		}
	}

	certSecretCopy.Data = map[string][]byte{
		"tls.crt": clientCert,
		"tls.key": clientKey,
	}

	// TODO: extend annotations to include more information about certs, including expiry date, etc.
	// HINT: take a look at openshift/service-serving-cert-signer for ideas.
	certSecretCopy.Annotations = map[string]string{
		annCertificateGenerated: "true",
	}

	// Check are Secrets different and perform update if they are.
	if !equality.Semantic.DeepEqual(certSecret, certSecretCopy) {
		_, err = c.kubeclientset.CoreV1().Secrets(certSecretCopy.Namespace).Update(certSecretCopy)
		if err != nil {
			// TODO: refactor event handling (hint: see #40).
			c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
			return err
		}
	}

	return nil
}
