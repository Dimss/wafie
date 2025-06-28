package controlplane

import (
	cwafv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func shouldSkipProtection(protection *cwafv1.Protection) bool {
	if protection.Application == nil {
		return true // skip if no application is associated
	}
	if len(protection.Application.Ingress) == 0 {
		return true // skip if no ingress is associated
	}
	return false
}

func newKubeClient() (*kubernetes.Clientset, error) {
	rc, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
