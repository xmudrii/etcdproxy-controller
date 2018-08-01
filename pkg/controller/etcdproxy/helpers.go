package etcdproxy

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
)

// newDeployment creates a new Deployment for a EtcdStorage resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the EtcdStorage resource that 'owns' it.
func newDeployment(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	etcdControllerNamespace, etcdProxyNamespace, etcdProxyImage,
	etcdCoreCAConfigMapName, etcdCoreCertSecretName string, etcdCoreURLs []string) *appsv1.Deployment {
	labels := map[string]string{
		"apiserver": etcdstorage.Name,
	}
	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName(etcdstorage),
			Namespace: etcdControllerNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "etcdproxy",
							Image:   etcdProxyImage,
							Command: []string{"/usr/local/bin/etcd", "grpc-proxy", "start"},
							Args: []string{
								flagfromString("endpoints", strings.Join(etcdCoreURLs, ",")),
								flagfromString("namespace", "/"+etcdProxyNamespace+"/"),
								"--listen-addr=0.0.0.0:2379",
								"--cacert=/etc/coreetcd-certs/ca/ca.crt",
								"--cert=/etc/coreetcd-certs/client/tls.crt",
								"--key=/etc/coreetcd-certs/client/tls.key",
								"--trusted-ca-file=/etc/etcdproxy-certs/ca/client-ca.crt",
								"--cert-file=/etc/etcdproxy-certs/server/tls.crt",
								"--key-file=/etc/etcdproxy-certs/server/tls.key",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "etcd",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 2379,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      etcdCoreCertSecretName,
									MountPath: "/etc/coreetcd-certs/client",
									ReadOnly:  true,
								},
								{
									Name:      etcdCoreCAConfigMapName,
									MountPath: "/etc/coreetcd-certs/ca",
									ReadOnly:  true,
								},
								{
									Name:      etcdProxyCAConfigMapName(etcdstorage),
									MountPath: "/etc/etcdproxy-certs/ca",
									ReadOnly:  true,
								},
								{
									Name:      etcdProxyServerCertsSecret(etcdstorage),
									MountPath: "/etc/etcdproxy-certs/server",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: etcdCoreCertSecretName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: etcdCoreCertSecretName,
								},
							},
						},
						{
							Name: etcdCoreCAConfigMapName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: etcdCoreCAConfigMapName,
									},
								},
							},
						},
						{
							Name: etcdProxyCAConfigMapName(etcdstorage),
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: etcdProxyCAConfigMapName(etcdstorage),
									},
								},
							},
						},
						{
							Name: etcdProxyServerCertsSecret(etcdstorage),
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: etcdProxyServerCertsSecret(etcdstorage),
								},
							},
						},
					},
				},
			},
		},
	}
}

func newService(etcdstorage *etcdstoragev1alpha1.EtcdStorage, etcdControllerNamespace string) *corev1.Service {
	labels := map[string]string{
		"apiserver": etcdstorage.Name,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName(etcdstorage),
			Namespace: etcdControllerNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       2379,
					TargetPort: intstr.FromInt(2379),
				},
			},
		},
	}
}

// deploymentName calculates name to be used to create a Deployment.
func deploymentName(etcdstorage *etcdstoragev1alpha1.EtcdStorage) string {
	return fmt.Sprintf("etcd-%s", etcdstorage.ObjectMeta.Name)
}

// serviceName calculates name to be used to create a Deployment.
func serviceName(etcdstorage *etcdstoragev1alpha1.EtcdStorage) string {
	return fmt.Sprintf("etcd-%s", etcdstorage.ObjectMeta.Name)
}

// etcdProxyCAConfigMapName calculates name to be used to create a etcdproxy CA ConfigMap.
func etcdProxyCAConfigMapName(etcdstorage *etcdstoragev1alpha1.EtcdStorage) string {
	return fmt.Sprintf("%s-ca-cert", etcdstorage.Name)
}

// etcdProxyServerCertsSecret calculates name to be used to create a etcdproxy server certs Secret.
func etcdProxyServerCertsSecret(etcdstorage *etcdstoragev1alpha1.EtcdStorage) string {
	return fmt.Sprintf("%s-server-cert", etcdstorage.Name)
}

// flagfromString returns double dash prefixed flag calculated from provided key and value.
func flagfromString(key, value string) string {
	return fmt.Sprintf("--%s=%s", key, value)
}

// ensureConfigMap ensures provided ConfigMap exists as it is provided. If ConfigMap is not found, it will be created.
func ensureConfigMap(kubeclientset kubernetes.Interface, required *corev1.ConfigMap) error {
	existing, err := kubeclientset.CoreV1().ConfigMaps(required.Namespace).Get(required.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = kubeclientset.CoreV1().ConfigMaps(required.Namespace).Create(required)
		return err
	}
	if err != nil {
		return err
	}

	// TODO: Anti-update hack. There is no appending in place, it will just override existing certificates. Appending will be added in #60.
	if _, ok := existing.Annotations[ProxyCertificateExpiryAnnotation]; ok {
		return nil
	}

	modified := false
	mergeStringMap(&modified, &existing.ObjectMeta.Annotations, required.ObjectMeta.Annotations)
	mergeStringMap(&modified, &existing.ObjectMeta.Labels, required.ObjectMeta.Labels)
	if equality.Semantic.DeepEqual(required.Data, existing.Data) && !modified {
		return nil
	}

	existing.Data = required.Data
	_, err = kubeclientset.CoreV1().ConfigMaps(existing.Namespace).Update(existing)

	return err
}

// ensureSecret ensures provided Secret exists as it is provided. If Secret is not found, it will be created.
func ensureSecret(kubeclientset kubernetes.Interface, required *corev1.Secret) error {
	existing, err := kubeclientset.CoreV1().Secrets(required.Namespace).Get(required.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = kubeclientset.CoreV1().Secrets(required.Namespace).Create(required)
		return err
	}
	if err != nil {
		return err
	}

	// TODO: Anti-update hack. There is no appending in place, it will just override existing certificates. Appending will be added in #60.
	if _, ok := existing.Annotations[ProxyCertificateExpiryAnnotation]; ok {
		return nil
	}

	modified := false
	mergeStringMap(&modified, &existing.ObjectMeta.Annotations, required.ObjectMeta.Annotations)
	mergeStringMap(&modified, &existing.ObjectMeta.Labels, required.ObjectMeta.Labels)
	if equality.Semantic.DeepEqual(required.Data, existing.Data) && !modified {
		return nil
	}

	existing.Data = required.Data
	_, err = kubeclientset.CoreV1().Secrets(existing.Namespace).Update(existing)

	return err
}

func mergeStringMap(modified *bool, destination *map[string]string, required map[string]string) {
	if *destination == nil {
		*destination = map[string]string{}
	}

	for k, v := range required {
		if destinationV, ok := (*destination)[k]; !ok || destinationV != v {
			(*destination)[k] = v
			*modified = true
		}
	}
}
