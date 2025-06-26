package controlplane

import (
	"context"
	"fmt"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"go.uber.org/zap"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type IngressPatcher struct {
	kc         *kubernetes.Clientset
	protection *cwafv1.Protection
	logger     *zap.Logger
}

func NewIngressPatcher(kc *kubernetes.Clientset, protection *cwafv1.Protection, logger *zap.Logger) *IngressPatcher {
	return &IngressPatcher{
		kc:         kc,
		protection: protection,
		logger:     logger.With(zap.String("host", protection.Application.Ingress[0].Host)),
	}
}

func (p *IngressPatcher) Patch() error {
	ingress, err := p.kc.NetworkingV1().
		Ingresses(p.protection.Application.Ingress[0].Namespace).
		Get(context.Background(), p.protection.Application.Ingress[0].Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if !p.shouldPatch(ingress) {
		return nil
	}
	if err := p.disableAppIngress(ingress); err != nil {
		return err
	}
	if err := p.createWafIngress(); err != nil {
		return err
	}
	return nil
}

func (p *IngressPatcher) disableAppIngress(ingress *v1.Ingress) error {
	if len(ingress.Spec.Rules) > 0 {
		ingress.Spec.Rules[0].Host = fmt.Sprintf("kguard-disabled-%s", p.protection.Application.Ingress[0].Host)
	}
	_, err := p.kc.NetworkingV1().Ingresses(p.protection.Application.Ingress[0].Namespace).
		Update(context.Background(), ingress, metav1.UpdateOptions{})
	return err
}

func (p *IngressPatcher) createWafIngress() error {
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("kguard-%s", p.protection.Application.Ingress[0].Name),
			Namespace:   p.protection.Application.Ingress[0].Namespace,
			Annotations: map[string]string{"kguard.io/owned": "true"},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: p.protection.Application.Ingress[0].Host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: nil,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: "wafy-core",
											Port: v1.ServiceBackendPort{
												Number: 8888,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := p.kc.
		NetworkingV1().
		Ingresses(p.protection.Application.Ingress[0].Namespace).
		Create(context.Background(), ingress, metav1.CreateOptions{})
	p.logger.Info("waffy ingress created")
	return err
}

func (p *IngressPatcher) shouldPatch(ingress *v1.Ingress) bool {
	if _, ok := ingress.ObjectMeta.Annotations["kguard.io/owned"]; ok {
		p.logger.Info("ingress already patched, skipping",
			zap.String("name", ingress.Name),
			zap.String("namespace", ingress.Namespace))
		return false
	}
	return true
}

func (p *IngressPatcher) Unpatch() error {
	return nil
}
