package ingresscache

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func getSvc(svcName, namespace string) (*v1.Service, error) {
	rc, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().
		Services(namespace).
		Get(context.TODO(),
			svcName,
			metav1.GetOptions{},
		)
}

func getSvcPortNumberBySvcPortName(portName, svcName, namespace string) (int32, error) {
	service, err := getSvc(svcName, namespace)
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

func getContainerPortBySvcPortNumber(portNumber int32, svcName, namespace string) (int32, error) {
	service, err := getSvc(svcName, namespace)
	if err != nil {
		return 0, err
	}
	for _, p := range service.Spec.Ports {
		if p.Port == portNumber {
			// if target port is set no further discovery is needed
			if p.TargetPort.IntVal != 0 {
				return p.TargetPort.IntVal, nil
			}
			// target port is not set,
			// need to query endpoint
			// to get actual port number
			endpointSlice, err := getEndpointSliceBySvcName(svcName, namespace)
			if err != nil {
				return 0, err
			}
			for _, port := range endpointSlice.Ports {
				if *port.Port == portNumber {
					return *port.Port, nil
				}
			}
			return p.Port, nil
		}
	}
	return 0, fmt.Errorf("can not find container port for service: %s", svcName)
}

func getContainerPortBySvcPortName(portName, svcName, namespace string) (int32, error) {
	service, err := getSvc(svcName, namespace)
	if err != nil {
		return 0, err
	}
	for _, p := range service.Spec.Ports {
		if p.Name == portName {
			// if target port is set to number, no further discovery is needed
			if p.TargetPort.IntVal != 0 {
				return p.TargetPort.IntVal, nil
			}
			// target port is set to name,
			// need to query endpoint
			// to get actual port number
			endpointSlice, err := getEndpointSliceBySvcName(svcName, namespace)
			if err != nil {
				return 0, err
			}
			for _, port := range endpointSlice.Ports {
				if *port.Name == portName {
					return *port.Port, nil
				}
			}
			return p.Port, nil
		}
	}
	return 0, fmt.Errorf("can not find container port for service: %s", svcName)
}

func getEndpointSliceBySvcName(svcName, namespace string) (*discoveryv1.EndpointSlice, error) {
	rc, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("kubernetes.io/service-name=%s", svcName)
	endpoints, err := clientset.DiscoveryV1().EndpointSlices(namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(endpoints.Items) == 0 {
		return nil, fmt.Errorf("no endpointslice found for service %s", svcName)
	}
	return &endpoints.Items[0], nil
}
