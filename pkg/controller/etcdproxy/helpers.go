/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcdproxy

import (
	"fmt"
	"io/ioutil"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
)

// newReplicaSet creates a new Deployment for a EtcdStorage resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the EtcdStorage resource that 'owns' it.
func newReplicaSet(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	etcdControllerNamespace, etcdProxyNamespace, etcdProxyImage,
	etcdCoreUrl, etcdCoreCAConfigMapName, etcdCoreCertSecretName string) *appsv1.ReplicaSet {
	labels := map[string]string{
		"controller": "epc",
	}
	replicas := int32(1)

	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      replicaSetName(etcdstorage),
			Namespace: etcdControllerNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
			},
		},
		Spec: appsv1.ReplicaSetSpec{
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
								flagfromString("endpoints", etcdCoreUrl),
								flagfromString("namespace", "/"+etcdProxyNamespace+"/"),
								"--listen-addr=0.0.0.0:2379",
								"--cacert=/etc/certs/ca/ca.pem",
								"--cert=/etc/certs/client/client.pem",
								"--key=/etc/certs/client/client-key.pem",
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
									MountPath: "/etc/certs/client",
									ReadOnly:  true,
								},
								{
									Name:      etcdCoreCAConfigMapName,
									MountPath: "/etc/certs/ca",
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
					},
				},
			},
		},
	}
}

func newService(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	etcdControllerNamespace string) *corev1.Service {
	labels := map[string]string{
		"controller": "epc",
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

// replicaSetName calculates name to be used to create a ReplicaSet.
func replicaSetName(etcdstorage *etcdstoragev1alpha1.EtcdStorage) string {
	return fmt.Sprintf("etcd-rs-%s", etcdstorage.ObjectMeta.Name)
}

// serviceName calculates name to be used to create a ReplicaSet.
func serviceName(etcdstorage *etcdstoragev1alpha1.EtcdStorage) string {
	return fmt.Sprintf("etcd-%s", etcdstorage.ObjectMeta.Name)
}

// flagfromString returns double dash prefixed flag calculated from provided key and value.
func flagfromString(key, value string) string {
	return fmt.Sprintf("--%s=%s", key, value)
}

// ControllerNamespace returns name of the namespace where controller is located. The namespace name is obtained
// from the "/var/run/secrets/kubernetes.io/serviceaccount/namespace" file. In case it's not possible to obtain it
// from that file, the function resorts to the default name, `kube-apiserver-storage`.
func ControllerNamespace(namespace string) (string, error) {
	if namespace != "" {
		return namespace, nil
	} else if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}

	return "", fmt.Errorf("unable to detect controller namespace")
}
