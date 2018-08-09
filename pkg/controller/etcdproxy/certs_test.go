package etcdproxy

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"time"

	"github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	"github.com/xmudrii/etcdproxy-controller/pkg/certs"
)

func TestEnsureServerCertificates(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: v1alpha1.EtdcStorageSpec{
				CACertConfigMaps: []v1alpha1.CABundleDestination{
					{
						Name:      "etcd-serving-ca",
						Namespace: "k8s-sample-apiserver",
					},
				},
				SigningCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ServingCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ClientCertificateValidity:  metav1.Duration{time.Hour * 24 * 60},
			},
		}
	}
	configMap := func(name, namespace string) *v1.ConfigMap {
		return &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}
	tests := []struct {
		name                string
		etcdProxyConfig     *EtcdProxyControllerConfig
		startingEtcdStorage *v1alpha1.EtcdStorage
		startingConfigMaps  []*v1.ConfigMap
	}{
		{
			name: "start with one serving configmap and add another",
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			startingEtcdStorage: etcdStorage("certs-test-1"),
			startingConfigMaps: []*v1.ConfigMap{
				configMap("etcd-serving-ca", "k8s-sample-apiserver"),
				configMap("etcd-serving-ca-2", "k8s-sample-apiserver"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testObjs := []runtime.Object{tc.startingEtcdStorage}
			for _, startingConfigMap := range tc.startingConfigMaps {
				testObjs = append(testObjs, startingConfigMap)
			}
			c := newEtcdProxyControllerMock(tc.etcdProxyConfig, testObjs)

			err := c.ensureServerCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			cm, err := c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMaps[0].Namespace).Get(tc.startingConfigMaps[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cm.Data["serving-ca.crt"]; !ok {
				t.Fatalf("expected serving certificate in configmap '%s/%s' but have not found it", cm.Namespace, cm.Name)
			}
			_, err = certs.ParseCertificateBytes([]byte(cm.Data["serving-ca.crt"]), nil)
			if err != nil {
				t.Fatal(err)
			}

			newDest := v1alpha1.CABundleDestination{
				Name:      tc.startingConfigMaps[1].Name,
				Namespace: tc.startingConfigMaps[1].Namespace,
			}
			tc.startingEtcdStorage.Spec.CACertConfigMaps = append(tc.startingEtcdStorage.Spec.CACertConfigMaps, newDest)
			err = c.ensureServerCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// Verify first.
			cm, err = c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMaps[0].Namespace).Get(tc.startingConfigMaps[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cm.Data["serving-ca.crt"]; !ok {
				t.Fatalf("expected serving certificate in configmap '%s/%s' but have not found it", cm.Namespace, cm.Name)
			}
			_, err = certs.ParseCertificateBytes([]byte(cm.Data["serving-ca.crt"]), nil)
			if err != nil {
				t.Fatal(err)
			}

			// Verify second.
			cm, err = c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMaps[1].Namespace).Get(tc.startingConfigMaps[1].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cm.Data["serving-ca.crt"]; !ok {
				t.Fatalf("expected serving certificate in configmap '%s/%s' but have not found it", cm.Namespace, cm.Name)
			}
			_, err = certs.ParseCertificateBytes([]byte(cm.Data["serving-ca.crt"]), nil)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEnsureServerCertificatesDuplication(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: v1alpha1.EtdcStorageSpec{
				CACertConfigMaps: []v1alpha1.CABundleDestination{
					{
						Name:      "etcd-serving-ca",
						Namespace: "k8s-sample-apiserver",
					},
				},
				SigningCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ServingCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ClientCertificateValidity:  metav1.Duration{time.Hour * 24 * 60},
			},
		}
	}
	configMap := func(name, namespace string) *v1.ConfigMap {
		return &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}
	tests := []struct {
		name                string
		etcdProxyConfig     *EtcdProxyControllerConfig
		startingEtcdStorage *v1alpha1.EtcdStorage
		startingConfigMaps  []*v1.ConfigMap
	}{
		{
			name: "ensure same server certificate will not be added two times to the bundle",
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			startingEtcdStorage: etcdStorage("certs-test-1"),
			startingConfigMaps: []*v1.ConfigMap{
				configMap("etcd-serving-ca", "k8s-sample-apiserver"),
				configMap("etcd-serving-ca-2", "k8s-sample-apiserver"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testObjs := []runtime.Object{tc.startingEtcdStorage}
			for _, startingConfigMap := range tc.startingConfigMaps {
				testObjs = append(testObjs, startingConfigMap)
			}
			c := newEtcdProxyControllerMock(tc.etcdProxyConfig, testObjs)

			// Generate certificate chain.
			err := c.ensureServerCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// Verify is number of certificates correct.
			cm, err := c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMaps[0].Namespace).Get(tc.startingConfigMaps[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cm.Data["serving-ca.crt"]; !ok {
				t.Fatalf("expected serving certificate in configmap '%s/%s' but have not found it", cm.Namespace, cm.Name)
			}
			crt, err := certs.ParseCertificateBytes([]byte(cm.Data["serving-ca.crt"]), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(crt.Certificates) != 2 {
				t.Fatalf("expected 2 certificates (ca + server) in the serving chain but got '%d'", len(crt.Certificates))
			}

			// Ensure is server certificate there. As certificate is still valid, this must not change number of certificates in the chain.
			err = c.ensureServerCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// Verify is number of certificates unchanged.
			cm, err = c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMaps[0].Namespace).Get(tc.startingConfigMaps[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cm.Data["serving-ca.crt"]; !ok {
				t.Fatalf("expected serving certificate in configmap '%s/%s' but have not found it", cm.Namespace, cm.Name)
			}
			crt, err = certs.ParseCertificateBytes([]byte(cm.Data["serving-ca.crt"]), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(crt.Certificates) != 2 {
				t.Fatalf("expected initial 2 certificates (ca + server) in the serving chain but additional certs were added (got '%d')", len(crt.Certificates))
			}
		})
	}
}

func TestEnsureClientCertificates(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: v1alpha1.EtdcStorageSpec{
				ClientCertSecrets: []v1alpha1.ClientCertificateDestination{
					{
						Name:      "etcd-client-cert",
						Namespace: "k8s-sample-apiserver",
					},
				},
				SigningCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ServingCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ClientCertificateValidity:  metav1.Duration{time.Hour * 24 * 60},
			},
		}
	}
	secret := func(name, namespace string) *v1.Secret {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: v1.SecretTypeTLS,
		}
	}
	tests := []struct {
		name                string
		etcdProxyConfig     *EtcdProxyControllerConfig
		startingEtcdStorage *v1alpha1.EtcdStorage
		startingSecrets     []*v1.Secret
	}{
		{
			name: "start with one client secret and add another",
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			startingEtcdStorage: etcdStorage("certs-test-1"),
			startingSecrets: []*v1.Secret{
				secret("etcd-client-cert", "k8s-sample-apiserver"),
				secret("etcd-client-cert-2", "k8s-sample-apiserver"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testObjs := []runtime.Object{tc.startingEtcdStorage}
			for _, startingSecret := range tc.startingSecrets {
				testObjs = append(testObjs, startingSecret)
			}
			c := newEtcdProxyControllerMock(tc.etcdProxyConfig, testObjs)

			err := c.ensureClientCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			certSecret, err := c.kubeclientset.CoreV1().Secrets(tc.startingSecrets[0].Namespace).Get(tc.startingSecrets[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := certSecret.Data["tls.crt"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			if _, ok := certSecret.Data["tls.key"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			_, err = certs.ParseCertificateBytes(certSecret.Data["tls.crt"], certSecret.Data["tls.key"])
			if err != nil {
				t.Fatal(err)
			}

			newDest := v1alpha1.ClientCertificateDestination{
				Name:      tc.startingSecrets[1].Name,
				Namespace: tc.startingSecrets[1].Namespace,
			}
			tc.startingEtcdStorage.Spec.ClientCertSecrets = append(tc.startingEtcdStorage.Spec.ClientCertSecrets, newDest)
			err = c.ensureClientCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// Verify first.
			certSecret, err = c.kubeclientset.CoreV1().Secrets(tc.startingSecrets[0].Namespace).Get(tc.startingSecrets[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := certSecret.Data["tls.crt"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			if _, ok := certSecret.Data["tls.key"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			_, err = certs.ParseCertificateBytes(certSecret.Data["tls.crt"], certSecret.Data["tls.key"])
			if err != nil {
				t.Fatal(err)
			}

			// Verify second.
			certSecret, err = c.kubeclientset.CoreV1().Secrets(tc.startingSecrets[1].Namespace).Get(tc.startingSecrets[1].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := certSecret.Data["tls.crt"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			if _, ok := certSecret.Data["tls.key"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			_, err = certs.ParseCertificateBytes(certSecret.Data["tls.crt"], certSecret.Data["tls.key"])
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEnsureClientCertificatesDuplication(t *testing.T) {
	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
		return &v1alpha1.EtcdStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: v1alpha1.EtdcStorageSpec{
				ClientCertSecrets: []v1alpha1.ClientCertificateDestination{
					{
						Name:      "etcd-client-cert",
						Namespace: "k8s-sample-apiserver",
					},
				},
				SigningCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ServingCertificateValidity: metav1.Duration{time.Hour * 24 * 60},
				ClientCertificateValidity:  metav1.Duration{time.Hour * 24 * 60},
			},
		}
	}
	secret := func(name, namespace string) *v1.Secret {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: v1.SecretTypeTLS,
		}
	}
	tests := []struct {
		name                string
		etcdProxyConfig     *EtcdProxyControllerConfig
		startingEtcdStorage *v1alpha1.EtcdStorage
		startingSecrets     []*v1.Secret
	}{
		{
			name: "ensure same client certificate will not be added two times to the bundle",
			etcdProxyConfig: &EtcdProxyControllerConfig{
				CoreEtcd: &CoreEtcdConfig{
					URLs:            []string{"https://test.etcd.svc:2379"},
					CAConfigMapName: "etcd-coreserving-ca",
					CertSecretName:  "etcd-coreserving-cert",
				},
				ControllerNamespace: "test-storage",
				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
			},
			startingEtcdStorage: etcdStorage("certs-test-1"),
			startingSecrets: []*v1.Secret{
				secret("etcd-client-cert", "k8s-sample-apiserver"),
				secret("etcd-client-cert-2", "k8s-sample-apiserver"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testObjs := []runtime.Object{tc.startingEtcdStorage}
			for _, startingSecret := range tc.startingSecrets {
				testObjs = append(testObjs, startingSecret)
			}
			c := newEtcdProxyControllerMock(tc.etcdProxyConfig, testObjs)

			// Generate the initial client certificate chain.
			err := c.ensureClientCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// Verify the number of certificate in the chain.
			certSecret, err := c.kubeclientset.CoreV1().Secrets(tc.startingSecrets[0].Namespace).Get(tc.startingSecrets[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := certSecret.Data["tls.crt"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			if _, ok := certSecret.Data["tls.key"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			crt, err := certs.ParseCertificateBytes(certSecret.Data["tls.crt"], certSecret.Data["tls.key"])
			if err != nil {
				t.Fatal(err)
			}
			if len(crt.Certificates) != 1 {
				t.Fatalf("expected one client certificate in the bundle but got '%d'", len(crt.Certificates))
			}

			// Run the generation loop once again.
			err = c.ensureClientCertificates(tc.startingEtcdStorage)
			if err != nil {
				t.Fatal(err)
			}

			// Verify the number of certificates, which should stay the same.
			certSecret, err = c.kubeclientset.CoreV1().Secrets(tc.startingSecrets[0].Namespace).Get(tc.startingSecrets[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := certSecret.Data["tls.crt"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			if _, ok := certSecret.Data["tls.key"]; !ok {
				t.Fatalf("expected client certificate in secret '%s/%s' but have not found it", certSecret.Namespace, certSecret.Name)
			}
			crt, err = certs.ParseCertificateBytes(certSecret.Data["tls.crt"], certSecret.Data["tls.key"])
			if err != nil {
				t.Fatal(err)
			}
			if len(crt.Certificates) != 1 {
				t.Fatalf("expected one client certificate in the bundle, but additional certificate got added (got '%d')", len(crt.Certificates))
			}
		})
	}
}
