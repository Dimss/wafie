package controlplane

import (
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"go.uber.org/zap"
	v1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
)

type IngressPatcher struct {
	kc         *kubernetes.Clientset
	proxyNs    string
	protection *cwafv1.Protection
	logger     *zap.Logger
}

func NewIngressPatcher(kc *kubernetes.Clientset, protection *cwafv1.Protection, proxyNs string, logger *zap.Logger) *IngressPatcher {
	return &IngressPatcher{
		kc:         kc,
		protection: protection,
		proxyNs:    proxyNs,
		logger:     logger.With(zap.String("host", protection.Application.Ingress[0].Host)),
	}
}

func (p *IngressPatcher) Patch() error {
	unprotectedIngress, err := p.getIngress(
		p.protection.Application.Ingress[0].Namespace,
		p.protection.Application.Ingress[0].Name,
	)
	// if app ingress is not found, create it
	if kerrors.IsNotFound(err) {
		if err := p.createdProtectedIngress(unprotectedIngress); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// if found, check if patch is needed
	if p.kguardOwned(unprotectedIngress) {
		return nil
	}
	// delete and recreate secure route
	if err := p.deleteIngress(unprotectedIngress); err != nil {
		return err
	}
	// mutate unprotected to protected ingress
	if err := p.createdProtectedIngress(unprotectedIngress); err != nil {
		return err
	}

	return nil
}

func (p *IngressPatcher) deleteIngress(ingress *v1.Ingress) error {
	if err := p.kc.
		NetworkingV1().
		Ingresses(ingress.Namespace).
		Delete(context.Background(), ingress.Name, metav1.DeleteOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (p *IngressPatcher) createdProtectedIngress(appIngress *v1.Ingress) error {
	protectedIngressRules := make([]v1.IngressRule, len(appIngress.Spec.Rules))
	for ruleIdx, appRule := range appIngress.Spec.Rules {
		protectedIngressPaths := make([]v1.HTTPIngressPath, len(appRule.HTTP.Paths))
		// Add WAF Paths
		for pathIdx, appPath := range appRule.HTTP.Paths {
			protectedIngressPaths[pathIdx] = v1.HTTPIngressPath{
				Path:     appPath.Path,
				PathType: appPath.PathType,
				Backend: v1.IngressBackend{
					Service: &v1.IngressServiceBackend{
						Name: "wafy-core",
						Port: v1.ServiceBackendPort{
							Number: 8888,
						},
					},
				},
			}
		}
		// Add WAF ingress rules
		protectedIngressRules[ruleIdx] = v1.IngressRule{
			Host: appRule.Host,
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: &v1.HTTPIngressRuleValue{
					Paths: protectedIngressPaths,
				},
			},
		}
	}
	protectedIngress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.protection.Application.Ingress[0].Name,
			Namespace:   p.proxyNs,
			Annotations: map[string]string{"kguard.io/owned": "true"},
		},
		Spec: v1.IngressSpec{
			Rules: protectedIngressRules,
		},
	}
	_, err := p.kc.
		NetworkingV1().
		Ingresses(p.protection.Application.Ingress[0].Namespace).
		Create(context.Background(), protectedIngress, metav1.CreateOptions{})
	p.logger.Info("kguard ingress created")
	return err
}

func (p *IngressPatcher) kguardOwned(ingress *v1.Ingress) bool {
	if _, ok := ingress.ObjectMeta.Annotations["kguard.io/owned"]; ok {
		p.logger.Info("ingress already patched, skipping",
			zap.String("name", ingress.Name),
			zap.String("namespace", ingress.Namespace))
		return true
	}
	return false
}

func (p *IngressPatcher) Unpatch() error {
	// check if unprotected ingress must be created
	createUnprotectedIngress := false
	if unprotectedIngress, err := p.getIngress(
		p.protection.Application.Ingress[0].Namespace,
		p.protection.Application.Ingress[0].Name,
	); kerrors.IsNotFound(err) {
		createUnprotectedIngress = true
	} else if err != nil {
		return err
	} else if p.kguardOwned(unprotectedIngress) { // in case waf and service ingress are in the name ns
		createUnprotectedIngress = true
	}
	// check if protected ingress must be deleted
	deleteProtectedIngress := true
	protectedIngress, err := p.getIngress(
		p.proxyNs,
		p.protection.Application.Ingress[0].Name)
	if kerrors.IsNotFound(err) {
		// protected ingress not found, do nothing
		deleteProtectedIngress = false
	} else if err != nil {
		return err
	} else if p.kguardOwned(protectedIngress) { // in case waf and service ingress are in the name ns
		deleteProtectedIngress = true
	}
	// delete protected ingress
	if deleteProtectedIngress {
		if err := p.deleteIngress(protectedIngress); err != nil {
			return err
		}
	}
	// create unprotected ingress
	if createUnprotectedIngress {
		// convert raw JSON ingress spec to v1.Ingress
		unprotectedIngress, err := p.rawJsonIngressToV1Ingress(
			p.protection.Application.Ingress[0].RawIngressSpec)
		if err != nil {
			return err
		}
		// create unprotected ingress
		if _, err := p.kc.NetworkingV1().
			Ingresses(unprotectedIngress.Namespace).
			Create(
				context.Background(),
				unprotectedIngress,
				metav1.CreateOptions{},
			); err != nil {
			return err
		}
	}
	return nil
}

func (p *IngressPatcher) rawJsonIngressToV1Ingress(rawSpec string) (*v1.Ingress, error) {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	options := json.SerializerOptions{
		Yaml:   false,
		Pretty: false,
		Strict: false,
	}
	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme,
		scheme,
		options,
	)
	obj := &unstructured.Unstructured{}
	if _, _, err := serializer.Decode(
		[]byte(rawSpec),
		&schema.GroupVersionKind{
			Group:   "",
			Version: "",
			Kind:    "",
		},
		obj,
	); err != nil {
		return nil, err
	}
	ingress := &v1.Ingress{}
	if err := runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(obj.Object, ingress); err != nil {
		return nil, err
	}
	ingress.Status = v1.IngressStatus{} // clear status to avoid conflicts
	ingress.ResourceVersion = ""        // clear resource version to avoid conflicts
	return ingress, nil
}

func (p *IngressPatcher) getIngress(namespace, name string) (*v1.Ingress, error) {
	return p.kc.NetworkingV1().
		Ingresses(namespace).
		Get(context.Background(), name, metav1.GetOptions{})
}
