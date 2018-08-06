package etcdproxy

import (
	"crypto/x509/pkix"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"github.com/xmudrii/etcdproxy-controller/pkg/certs"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

// ProxyCertificateExpiryAnnotation is annotating are certificates successfully created and stored in the Secret or ConfigMap.
const ProxyCertificateExpiryAnnotation = "etcd.xmudrii.com/certificate-generated"

// ensureClientCertificates ensures there are client CA bundle and client certificates generated and in place.
// The Client CA bundle is stored in the ConfigMap in the controller namespace.
// The Client certificates are stored in Secrets defined in the EtcdStorage Spec.
func (c *EtcdProxyController) ensureClientCertificates(etcdstorage *etcdstoragev1alpha1.EtcdStorage) error {
	var clientCA *certs.Certificate
	var errs []error
	for _, clientCertSecret := range etcdstorage.Spec.ClientCertSecrets {
		secretClientCert, err := c.kubeclientset.CoreV1().Secrets(clientCertSecret.Namespace).Get(clientCertSecret.Name, metav1.GetOptions{})
		// TODO: This is hack and we should get rid of it.
		if errors.IsNotFound(err) {
			secretClientCert, err = c.kubeclientset.CoreV1().Secrets(clientCertSecret.Namespace).Create(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: clientCertSecret.Name, Namespace: clientCertSecret.Namespace}})
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if expiry, ok := secretClientCert.Annotations[ProxyCertificateExpiryAnnotation]; ok {
			certExpiry, err := time.Parse(time.RFC3339, expiry)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if certExpiry.After(time.Now()) {
				continue
			}
		}

		// Generate CA.
		// TODO: Simplify this.
		if clientCA == nil {
			clientCAConfigMap, err := c.kubeclientset.CoreV1().ConfigMaps(c.config.ControllerNamespace).Get(etcdProxyCAConfigMapName(etcdstorage), metav1.GetOptions{})
			// TODO: This is hack and we should get rid of it.
			if errors.IsNotFound(err) {
				clientCAConfigMap, err = c.kubeclientset.CoreV1().ConfigMaps(c.config.ControllerNamespace).Create(&v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: etcdProxyCAConfigMapName(etcdstorage), Namespace: c.config.ControllerNamespace}})
			}
			if err != nil {
				errs = append(errs, err)
				continue
			}
			clientCA, err = c.generateClientCACertificate(etcdstorage)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if clientCABytes, ok := clientCAConfigMap.Data["client-ca.crt"]; ok {
				oldClientCA, err := certs.ParseCertificateBytes([]byte(clientCABytes), nil)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				clientCA.Certificates = append(clientCA.Certificates, oldClientCA.Certificates...)
			}
			clientCA.Certificates = certs.FilterExpiredCerts(clientCA.Certificates...)
			err = c.updateClientBundleConfigMap(etcdstorage, clientCA)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}

		clientCert, err := c.generateClientCertificate(etcdstorage, clientCA, clientCertSecret)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = c.updateClientCertSecret(etcdstorage, clientCertSecret, clientCert)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// ensureServerCertificates ensures there are serving CA bundle and server certificates generated and in place.
// The Serving CA bundle is stored in the ConfigMaps defined in the EtcdStorage Spec.
// The Server certificates are stored in Secrets in the controller namespace.
func (c *EtcdProxyController) ensureServerCertificates(etcdstorage *etcdstoragev1alpha1.EtcdStorage) error {
	serverSecret, err := c.kubeclientset.CoreV1().Secrets(c.config.ControllerNamespace).Get(etcdProxyServerCertsSecret(etcdstorage), metav1.GetOptions{})
	// TODO: This is hack and we should get rid of it.
	if errors.IsNotFound(err) {
		serverSecret, err = c.kubeclientset.CoreV1().Secrets(c.config.ControllerNamespace).Create(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: etcdProxyServerCertsSecret(etcdstorage), Namespace: c.config.ControllerNamespace}})
	}
	if err != nil {
		return err
	}

	if expiry, ok := serverSecret.Annotations[ProxyCertificateExpiryAnnotation]; ok {
		certExpiry, err := time.Parse(time.RFC3339, expiry)
		if err != nil {
			return err
		}
		if certExpiry.After(time.Now()) {
			return nil
		}
	}

	servingCA, serverCert, err := c.generateServerBundle(etcdstorage)
	if err != nil {
		return err
	}
	err = c.updateServerCertSecret(etcdstorage, serverCert)
	if err != nil {
		return err
	}

	var errs []error
	for _, cm := range etcdstorage.Spec.CACertConfigMaps {
		configMap, err := c.kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		var ca *certs.Certificate
		if oldCABytes, ok := configMap.Data["serving-ca.crt"]; ok {
			ca, err = certs.ParseCertificateBytes([]byte(oldCABytes), nil)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
		// TODO: This is *very* ugly.
		if ca != nil {
			ca.Certificates = append(ca.Certificates, servingCA.Certificates...)
		} else {
			ca = servingCA
		}
		ca.Certificates = certs.FilterExpiredCerts(ca.Certificates...)
		if err != nil {
			return err
		}
		err = c.updateServingBundleConfigMap(etcdstorage, cm, ca)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// TODO: Potentially, we don't need that many helpers. Maybe moving some of them to helpers.go is good idea as well.
// generateClientBundle generates etcd-proxy Client CA bundle.
func (c *EtcdProxyController) generateClientCACertificate(etcdstorage *etcdstoragev1alpha1.EtcdStorage) (*certs.Certificate, error) {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))
	serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(etcdstorage), c.config.ControllerNamespace)

	// Generate the Client CA bundle.
	return certs.NewCACertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-client-signer-%v", serviceUrl, time.Now().Unix()),
	}, r.Int63n(100000), currentTime)
}

// generateClientBundle generates etcd-proxy client certificate and key pair based on provided Client CA bundle.
func (c *EtcdProxyController) generateClientCertificate(etcdstorage *etcdstoragev1alpha1.EtcdStorage, clientCABundle *certs.Certificate, clientCertSecret etcdstoragev1alpha1.ClientCertificateDestination) (*certs.Certificate, error) {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))

	return clientCABundle.NewClientCertificate(pkix.Name{CommonName: fmt.Sprintf("client-%s-%s", clientCertSecret.Namespace, clientCertSecret.Name)},
		r.Int63n(100000), currentTime)
}

// generateServerBundle generates etcd-proxy Serving CA bundle and Server certificate.
func (c *EtcdProxyController) generateServerBundle(etcdstorage *etcdstoragev1alpha1.EtcdStorage) (*certs.Certificate, *certs.Certificate, error) {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))
	serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(etcdstorage), c.config.ControllerNamespace)

	// Generate the Serving CA bundle.
	servingCA, err := certs.NewCACertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-server-signer-%v", serviceUrl, time.Now().Unix()),
	}, r.Int63n(100000), currentTime)
	if err != nil {
		return nil, nil, err
	}

	// Generate server certificate/key pair.
	serverCerts, err := servingCA.NewServerCertificate(pkix.Name{CommonName: serviceUrl},
		[]string{serviceUrl}, r.Int63n(100000), currentTime)
	if err != nil {
		return nil, nil, err
	}

	return servingCA, serverCerts, nil
}

// updateClientBundleConfigMap ensures the ConfigMap contains the required Client CA bundle.
// The Client CA bundle is located in the controller namespace and is used by the etcd-proxy to verify API server identity.
func (c *EtcdProxyController) updateClientBundleConfigMap(etcdstorage *etcdstoragev1alpha1.EtcdStorage, clientCABundle *certs.Certificate) error {
	clientCABytes, _, err := clientCABundle.GetPEMBytes()
	if err != nil {
		return err
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdProxyCAConfigMapName(etcdstorage),
			Namespace: c.config.ControllerNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Data: map[string]string{
			"client-ca.crt": string(clientCABytes),
		},
	}

	return ensureConfigMap(c.kubeclientset, configMap)
}

// updateServingBundleConfigMap ensures the ConfigMap contains the required Serving CA bundle.
// The Serving CA bundle is located in the API server namespace and is used by the API server to verify etcd-proxy identity.
func (c *EtcdProxyController) updateServingBundleConfigMap(etcdstorage *etcdstoragev1alpha1.EtcdStorage, caDestination etcdstoragev1alpha1.CABundleDestination, servingCABundle *certs.Certificate) error {
	servingCABytes, _, err := servingCABundle.GetPEMBytes()
	if err != nil {
		return err
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caDestination.Name,
			Namespace: caDestination.Namespace,
		},
		Data: map[string]string{
			"serving-ca.crt": string(servingCABytes),
		},
	}

	return ensureConfigMap(c.kubeclientset, configMap)
}

// updateServerCertSecret ensures the Secret contains the required Server Certificate and Key.
// The server certificate is used by etcd-proxy.
func (c *EtcdProxyController) updateServerCertSecret(etcdstorage *etcdstoragev1alpha1.EtcdStorage, serverCert *certs.Certificate) error {
	serverCertBytes, serverCertKey, err := serverCert.GetPEMBytes()
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdProxyServerCertsSecret(etcdstorage),
			Namespace: c.config.ControllerNamespace,
			Annotations: map[string]string{
				ProxyCertificateExpiryAnnotation: serverCert.Certificates[0].NotAfter.Format(time.RFC3339),
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": serverCertBytes,
			"tls.key": serverCertKey,
		},
	}

	return ensureSecret(c.kubeclientset, secret)
}

// updateClientCertSecret ensures the Secret contains the required Client Certificate and Key.
// The client certificate is used by the API server to authenticate with etcd-proxy.
func (c *EtcdProxyController) updateClientCertSecret(etcdstorage *etcdstoragev1alpha1.EtcdStorage, certDestination etcdstoragev1alpha1.ClientCertificateDestination, clientCert *certs.Certificate) error {
	clientCertBytes, clientKeyBytes, err := clientCert.GetPEMBytes()
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certDestination.Name,
			Namespace: certDestination.Namespace,
			Annotations: map[string]string{
				ProxyCertificateExpiryAnnotation: clientCert.Certificates[0].NotAfter.Format(time.RFC3339),
			},
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": clientCertBytes,
			"tls.key": clientKeyBytes,
		},
	}

	return ensureSecret(c.kubeclientset, secret)
}
