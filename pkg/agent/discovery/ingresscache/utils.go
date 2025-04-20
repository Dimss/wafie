package ingresscache

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func getSvcPortByName(portName, svcName, namespace string) (int32, error) {
	rc, err := config.GetConfig()
	if err != nil {
		return 0, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return 0, err
	}
	service, err := clientset.CoreV1().
		Services(namespace).
		Get(context.TODO(), svcName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	for _, p := range service.Spec.Ports {
		if p.Name == portName {
			return p.Port, nil
		}
	}
	return 0, nil
}
