package etcdproxy

//import (
//	"crypto/x509/pkix"
//	"fmt"
//	"math/rand"
//	"testing"
//	"time"
//
//	"k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//
//	"github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
//	"github.com/xmudrii/etcdproxy-controller/pkg/certs"
//)
//
//func TestAppendClientCACert(t *testing.T) {
//	configMap := func(name, namespace string, data map[string]string) *v1.ConfigMap {
//		return &v1.ConfigMap{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      name,
//				Namespace: namespace,
//			},
//			Data: data,
//		}
//	}
//	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
//		return &v1alpha1.EtcdStorage{
//			ObjectMeta: metav1.ObjectMeta{Name: name},
//		}
//	}
//	tests := []struct {
//		name                string
//		startingEtcdStorage *v1alpha1.EtcdStorage
//		startingConfigMap   *v1.ConfigMap
//		etcdProxyConfig     *EtcdProxyControllerConfig
//		expectedCACertNum   int
//	}{
//		{
//			name:                "append two client ca certs to empty configmap",
//			startingEtcdStorage: etcdStorage("test-1"),
//			startingConfigMap:   configMap("test-1-ca-cert", "kube-apiserver-storage", nil),
//			etcdProxyConfig: &EtcdProxyControllerConfig{
//				CoreEtcd: &CoreEtcdConfig{
//					URLs:            []string{"https://test.etcd.svc:2379"},
//					CAConfigMapName: "etcd-coreserving-ca",
//					CertSecretName:  "etcd-coreserving-cert",
//				},
//				ControllerNamespace: "kube-apiserver-storage",
//				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
//			},
//			expectedCACertNum: 2,
//		},
//	}
//
//	for _, tc := range tests {
//		// EtcdProxyController init.
//		testObjs := []runtime.Object{}
//		if tc.startingConfigMap != nil {
//			testObjs = append(testObjs, tc.startingConfigMap)
//		}
//		c := newEtcdProxyControllerMock(tc.etcdProxyConfig, []runtime.Object{tc.startingEtcdStorage}, testObjs)
//
//		serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(tc.startingEtcdStorage), c.config.ControllerNamespace)
//		for i := 0; i < tc.expectedCACertNum; i++ {
//			currentTime := time.Now
//			r := rand.New(rand.NewSource(currentTime().UnixNano()))
//
//			// Generate first Client CA cert and append it to the empty bundle.
//			serverSigner, err := certs.NewCACertificate(pkix.Name{CommonName: fmt.Sprintf("%s-server-signer-%v", serviceUrl, time.Now().Unix())}, r.Int63n(100000), currentTime)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			// Append first cert to the empty CA bundle.
//			err = c.appendClientCACert(tc.startingEtcdStorage, serverSigner)
//			if err != nil {
//				t.Fatal(err)
//			}
//		}
//
//		// Verify number of certificates.
//		cm, err := c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMap.Namespace).Get(tc.startingConfigMap.Name, metav1.GetOptions{})
//		if err != nil {
//			t.Fatal(err)
//		}
//		if _, ok := cm.Data["client-ca.crt"]; !ok {
//			t.Fatal("'clinet-ca.crt' file not present in the configmap")
//		}
//		ca, err := certs.ParseCertificateBytes([]byte(cm.Data["client-ca.crt"]), nil)
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		if len(ca.Certificates) != tc.expectedCACertNum {
//			t.Fatalf("expected '%d' certificate in the bundle but got '%d'", tc.expectedCACertNum, len(ca.Certificates))
//		}
//	}
//}
//
//func TestAppendServingCACert(t *testing.T) {
//	configMap := func(name, namespace string, data map[string]string) *v1.ConfigMap {
//		return &v1.ConfigMap{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      name,
//				Namespace: namespace,
//			},
//			Data: data,
//		}
//	}
//	etcdStorage := func(name string) *v1alpha1.EtcdStorage {
//		return &v1alpha1.EtcdStorage{
//			ObjectMeta: metav1.ObjectMeta{Name: name},
//			Spec: v1alpha1.EtdcStorageSpec{
//				CACertConfigMaps: []v1alpha1.CABundleDestination{
//					{
//						Name:      "etcd-serving-ca",
//						Namespace: "kube-apiserver-storage",
//					},
//				},
//			},
//		}
//	}
//	tests := []struct {
//		name                string
//		startingEtcdStorage *v1alpha1.EtcdStorage
//		startingConfigMap   *v1.ConfigMap
//		etcdProxyConfig     *EtcdProxyControllerConfig
//		expectedCACertNum   int
//	}{
//		{
//			name:                "append two serving ca certs to empty configmap",
//			startingEtcdStorage: etcdStorage("test-1"),
//			startingConfigMap:   configMap("etcd-serving-ca", "kube-apiserver-storage", nil),
//			etcdProxyConfig: &EtcdProxyControllerConfig{
//				CoreEtcd: &CoreEtcdConfig{
//					URLs:            []string{"https://test.etcd.svc:2379"},
//					CAConfigMapName: "etcd-coreserving-ca",
//					CertSecretName:  "etcd-coreserving-cert",
//				},
//				ControllerNamespace: "kube-apiserver-storage",
//				ProxyImage:          "quay.io/coreos/etcd:v3.2.18",
//			},
//			expectedCACertNum: 2,
//		},
//	}
//
//	for _, tc := range tests {
//		// EtcdProxyController init.
//		testObjs := []runtime.Object{}
//		if tc.startingConfigMap != nil {
//			testObjs = append(testObjs, tc.startingConfigMap)
//		}
//		c := newEtcdProxyControllerMock(tc.etcdProxyConfig, []runtime.Object{tc.startingEtcdStorage}, testObjs)
//
//		serviceUrl := fmt.Sprintf("%s.%s.svc", serviceName(tc.startingEtcdStorage), c.config.ControllerNamespace)
//		for i := 0; i < tc.expectedCACertNum; i++ {
//			currentTime := time.Now
//			r := rand.New(rand.NewSource(currentTime().UnixNano()))
//
//			// Generate first Client CA cert and append it to the empty bundle.
//			serverSigner, err := certs.NewCACertificate(pkix.Name{CommonName: fmt.Sprintf("%s-server-signer-%v", serviceUrl, time.Now().Unix())}, r.Int63n(100000), currentTime)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			// Append first cert to the empty CA bundle.
//			err = c.appendServingCACert(tc.startingEtcdStorage, serverSigner)
//			if err != nil {
//				t.Fatal(err)
//			}
//		}
//
//		// Verify number of certificates.
//		cm, err := c.kubeclientset.CoreV1().ConfigMaps(tc.startingConfigMap.Namespace).Get(tc.startingConfigMap.Name, metav1.GetOptions{})
//		if err != nil {
//			t.Fatal(err)
//		}
//		if _, ok := cm.Data["serving-ca.crt"]; !ok {
//			t.Fatal("'serving-ca.crt' file not present in the configmap")
//		}
//		ca, err := certs.ParseCertificateBytes([]byte(cm.Data["serving-ca.crt"]), nil)
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		if len(ca.Certificates) != tc.expectedCACertNum {
//			t.Fatalf("expected '%d' certificate in the bundle but got '%d'", tc.expectedCACertNum, len(ca.Certificates))
//		}
//	}
//}
