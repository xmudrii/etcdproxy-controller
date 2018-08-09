package etcdproxy

import (
	"crypto/x509/pkix"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"github.com/xmudrii/etcdproxy-controller/pkg/certs"
)

const (
	// ProxyCertificateExpiryAnnotation contains the certificate expiration date in RFC3339 format.
	ProxyCertificateExpiryAnnotation = "etcd.xmudrii.com/certificate-expiry-date"
	// ProxyCertificateSignedBy contains the common name of the certificate that signed another certificate.
	ProxyCertificateSignedBy = "etcd.xmudrii.com/certificate-signed-by"
)

// ensureClientCertificates handles certificate generating, renewal and rotation for Client CA bundle and Client certificates.
// The Client CA bundle is saved in a ConfigMap located in the controller namespace. The ConfigMap is named etcdstorageName-ca-cert.
// The Client certificate/key pairs are stored in the Secrets defined in the EtcdStorage Spec.
// The EtcdProxy controller assumes Secrets for Client certificates are already created, but if not, the controller will try to create them.
// Creating Secrets for Client certificates requires the appropriate RBAC roles if RBAC is enabled on cluster.
//
// This function reads all Secrets provided in the EtcdStorage Spec and checks is the 'etcd.xmudrii.com/certificate-expiry-date'
// annotation present and contains valid date. If date is in the past, i.e. certificate is expired, or the annotation is not present controller:
// * Generates new CA certificate. If CA bundle already exists in the controller namespace, the new CA certificate will be appended to the bundle.
// Expired CA certificates from the bundle are removed in this phase.
// * Generates new Client certificate/key pair using the newly generated CA certificate and updates the appropriate Secret with new pair.
// TODO: the API server have to be "restarted" manually to pick up changes. Hopefully, this to be fixed in future Kube versions.
func (c *EtcdProxyController) ensureClientCertificates(etcdstorage *etcdstoragev1alpha1.EtcdStorage) error {
	var signingCertKeyPair *certs.Certificate
	var errs []error
	for _, clientCertSecret := range etcdstorage.Spec.ClientCertSecrets {
		// Get Secret from Kube if it exists or return new, empty, Secret.
		secret, err := c.kubeclientset.CoreV1().Secrets(clientCertSecret.Namespace).Get(clientCertSecret.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			secret = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        clientCertSecret.Name,
					Namespace:   clientCertSecret.Namespace,
					Annotations: map[string]string{},
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{},
			}
			err = nil
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Check is annotation containing expiry date present and valid. If it is valid, we're skipping this iteration.
		if expiry, ok := secret.Annotations[ProxyCertificateExpiryAnnotation]; ok {
			certExpiry, err := time.Parse(time.RFC3339, expiry)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			// If Certificate is not-expired, skip this iteration.
			if certExpiry.After(time.Now()) {
				continue
			}
		}

		// Generate Client CA certificate and append it to the bundle.
		// The bundle is located in the ConfigMap in the controller namespace.
		// We're generating one Client CA in this loop for all Client certificates that are going to be regenerated in this loop/iteration.
		// TODO: Decide is this a good idea. Another idea is to genearte new CA certificate for each Client certificate that has to be regenrated.
		// TODO: The negative side in this case is that we could have to restart etcd-proxy for each certificate appended and to get/update for each one instead of once.
		if signingCertKeyPair == nil {
			// Generate new Client CA certificate.
			signingCertKeyPair, err = c.generateClientSigningCertKeyPair(etcdstorage)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			// Get Client CA ConfigMap and check does it contain CA bundle. If yes, append new client CA certificate to the bundle.
			clientCAConfigMap, err := c.kubeclientset.CoreV1().ConfigMaps(c.config.ControllerNamespace).Get(etcdProxyCAConfigMapName(etcdstorage), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				clientCAConfigMap = &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        etcdProxyCAConfigMapName(etcdstorage),
						Namespace:   c.config.ControllerNamespace,
						Annotations: map[string]string{},
					},
					Data: map[string]string{},
				}
				err = nil
			}
			if err != nil {
				errs = append(errs, err)
			}

			if clientCABytes, ok := clientCAConfigMap.Data["client-ca.crt"]; ok {
				oldClientCA, err := certs.ParseCertificateBytes([]byte(clientCABytes), nil)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				signingCertKeyPair.Certificates = append(signingCertKeyPair.Certificates, oldClientCA.Certificates...)
			}
			// Filter expired certificates in the Client CA bundle.
			signingCertKeyPair.Certificates = certs.FilterExpiredCerts(signingCertKeyPair.Certificates...)

			// Update ConfigMap with the updated Client CA bundle. If ConfigMap doesn't exist, it will be created.
			clientCABytes, _, err := signingCertKeyPair.GetPEMBytes()
			if err != nil {
				return err
			}
			clientCAConfigMap.Data = map[string]string{
				"client-ca.crt": string(clientCABytes),
			}

			err = ensureConfigMap(c.kubeclientset, clientCAConfigMap)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}

		// Generate new Client certificate/key pair using the newly-generated Client CA certificate and update
		// the appropriate Secret.
		clientCert, err := c.generateClientCertificate(etcdstorage, signingCertKeyPair, clientCertSecret)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		clientCertBytes, clientKeyBytes, err := clientCert.GetPEMBytes()
		if err != nil {
			return err
		}

		// Build new Secret and apply it.
		secret.Annotations = map[string]string{
			ProxyCertificateExpiryAnnotation: clientCert.Certificates[0].NotAfter.Format(time.RFC3339),
			ProxyCertificateSignedBy:         signingCertKeyPair.Certificates[0].Issuer.CommonName,
		}
		secret.Data = map[string][]byte{
			"tls.crt": clientCertBytes,
			"tls.key": clientKeyBytes,
		}

		err = ensureSecret(c.kubeclientset, secret)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// ensureServerCertificates handles certificate generating, renewal and rotation for Serving CA bundle and Server certificates.
// The Serving CA bundle is saved in a ConfigMaps defined in EtcdStorage Spec.
// The Server certificate/key pair is stored in the Secrets named etcdstorageName-server-cert in the controller namespace.
// The EtcdProxy controller assumes ConfigMaps for the Serving CA bundle are already created, but if not, the controller will try to create them.
// Creating ConfigMaps for storing the Serving CA bundle requires the appropriate RBAC roles if RBAC is enabled on cluster.
//
// This function reads the Secret in the controller namespace and checks is the 'etcd.xmudrii.com/certificate-expiry-date'
// annotation present and contains valid date. If date is in the past, i.e. certificate is expired, or the annotation is not present controller:
// * Generates new CA certificate. The new CA certificate is appended to all ConfigMaps specified by the EtcdStorage Spec.
// Expired CA certificates from the bundle are removed in this phase.
// * Generates new Server certificate/key pair using the newly generated CA certificate and update Secret in the controller namespace with new pair.
// * "Restarts" etcd-proxy to pick-up new changes.
// TODO: Implement "restarting" etcd-proxy. This is going to be done using rolling updates.
func (c *EtcdProxyController) ensureServerCertificates(etcdstorage *etcdstoragev1alpha1.EtcdStorage) error {
	serverSecret, err := c.kubeclientset.CoreV1().Secrets(c.config.ControllerNamespace).Get(etcdProxyServerCertsSecret(etcdstorage), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		serverSecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        etcdProxyServerCertsSecret(etcdstorage),
				Namespace:   c.config.ControllerNamespace,
				Annotations: map[string]string{},
			},
			Type: v1.SecretTypeTLS,
			Data: map[string][]byte{},
		}
		err = nil
	}
	if err != nil {
		return err
	}

	var serverCert *certs.Certificate
	// Check does the annotation contain valid expiry date, and if not, or it is expired, regenerate the certificate.
	if expiry, ok := serverSecret.Annotations[ProxyCertificateExpiryAnnotation]; ok {
		certExpiry, err := time.Parse(time.RFC3339, expiry)
		if err != nil {
			return err
		}
		if certExpiry.Before(time.Now()) {
			serverCert, err = c.generateServerBundle(etcdstorage)
			if err != nil {
				return err
			}

			// Write new Server certificate/key pair to the Secret in the controller namespace.
			serverCertBytes, serverKeyBytes, err := serverCert.GetPEMBytes()
			if err != nil {
				return err
			}

			serverSecret.Annotations = map[string]string{
				ProxyCertificateExpiryAnnotation: serverCert.Certificates[0].NotAfter.Format(time.RFC3339),
				ProxyCertificateSignedBy:         serverCert.Certificates[0].Issuer.CommonName,
			}
			serverSecret.Data = map[string][]byte{
				"tls.crt": serverCertBytes,
				"tls.key": serverKeyBytes,
			}
			err = ensureSecret(c.kubeclientset, serverSecret)
			if err != nil {
				return err
			}
		}
	} else if !ok { // If the annotation is not present, generate the certificate.
		serverCert, err = c.generateServerBundle(etcdstorage)
		if err != nil {
			return err
		}

		// Write new Server certificate/key pair to the Secret in the controller namespace.
		serverCertBytes, serverKeyBytes, err := serverCert.GetPEMBytes()
		if err != nil {
			return err
		}

		serverSecret.Annotations = map[string]string{
			ProxyCertificateExpiryAnnotation: serverCert.Certificates[0].NotAfter.Format(time.RFC3339),
			ProxyCertificateSignedBy:         serverCert.Certificates[0].Issuer.CommonName,
		}
		serverSecret.Data = map[string][]byte{
			"tls.crt": serverCertBytes,
			"tls.key": serverKeyBytes,
		}
		err = ensureSecret(c.kubeclientset, serverSecret)
		if err != nil {
			return err
		}
	}

	if serverCert == nil {
		serverCert, err = certs.ParseCertificateBytes(serverSecret.Data["tls.crt"], serverSecret.Data["tls.key"])
		if err != nil {
			return err
		}
	}

	// Append new Serving CA certificate to the bundle in all ConfigMaps defined by EtcdStorage Spec.
	var errs []error
	for _, cm := range etcdstorage.Spec.CACertConfigMaps {
		// Get CA bundle from the ConfigMap, check does it already have certificates in the bundle, append new one to it,
		// and filter expired certificates.
		configMap, err := c.kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			configMap = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        cm.Name,
					Namespace:   cm.Namespace,
					Annotations: map[string]string{},
				},
				Data: map[string]string{},
			}
			err = nil
		}
		if err != nil {
			return err
		}

		if signedBy, ok := configMap.Annotations[ProxyCertificateSignedBy]; ok {
			if signedBy == serverCert.Certificates[0].Issuer.CommonName {
				continue
			}
		}

		var ca *certs.Certificate
		if oldCABytes, ok := configMap.Data["serving-ca.crt"]; ok {
			ca, err = certs.ParseCertificateBytes([]byte(oldCABytes), nil)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}

		if ca != nil {
			ca.Certificates = append(ca.Certificates, serverCert.Certificates...)
		} else {
			ca = serverCert
		}

		// Filter expired certificates in the Serving CA bundle.
		ca.Certificates = certs.FilterExpiredCerts(ca.Certificates...)
		if err != nil {
			return err
		}
		// Update appropriate ConfigMap with the new Serving CA bundle.
		servingCABytes, _, err := ca.GetPEMBytes()
		if err != nil {
			return err
		}
		configMap.Annotations = map[string]string{
			ProxyCertificateSignedBy: serverCert.Certificates[0].Issuer.CommonName,
		}
		configMap.Data = map[string]string{
			"serving-ca.crt": string(servingCABytes),
		}
		err = ensureConfigMap(c.kubeclientset, configMap)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// generateClientBundle generates new etcd-proxy Client CA bundle.
func (c *EtcdProxyController) generateClientSigningCertKeyPair(etcdstorage *etcdstoragev1alpha1.EtcdStorage) (*certs.Certificate, error) {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))
	serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(etcdstorage), c.config.ControllerNamespace)

	// Generate the Client CA bundle.
	return certs.NewCACertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-client-signer-%v", serviceUrl, time.Now().Unix()),
	}, r.Int63n(100000), currentTime)
}

// generateClientBundle generates new etcd-proxy client certificate/key pair based on provided Client CA bundle.
func (c *EtcdProxyController) generateClientCertificate(etcdstorage *etcdstoragev1alpha1.EtcdStorage, clientCABundle *certs.Certificate, clientCertSecret etcdstoragev1alpha1.ClientCertificateDestination) (*certs.Certificate, error) {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))

	return clientCABundle.NewClientCertificate(pkix.Name{CommonName: fmt.Sprintf("client-%s-%s", clientCertSecret.Namespace, clientCertSecret.Name)},
		r.Int63n(100000), currentTime)
}

// generateServerBundle generates both Serving CA bundle and Server certificate/key pair.
func (c *EtcdProxyController) generateServerBundle(etcdstorage *etcdstoragev1alpha1.EtcdStorage) (*certs.Certificate, error) {
	currentTime := time.Now
	r := rand.New(rand.NewSource(currentTime().UnixNano()))
	serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(etcdstorage), c.config.ControllerNamespace)

	// Generate the Serving CA bundle.
	servingCA, err := certs.NewCACertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-server-signer-%v", serviceUrl, time.Now().Unix()),
	}, r.Int63n(100000), currentTime)
	if err != nil {
		return nil, err
	}

	// Generate server certificate/key pair.
	serverCerts, err := servingCA.NewServerCertificate(pkix.Name{
		CommonName: fmt.Sprintf("%s-serving-cert-%v", serviceUrl, time.Now().Unix()),
	}, []string{serviceUrl}, r.Int63n(100000), currentTime)
	if err != nil {
		return nil, err
	}

	return serverCerts, nil
}
