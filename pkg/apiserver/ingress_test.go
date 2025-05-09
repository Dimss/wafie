package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateIngressWithNoneExistingApp(t *testing.T) {
	svc := NewIngressService(applogger.NewLogger())
	req := connect.NewRequest(
		&cwafv1.CreateIngressRequest{
			Ingress: &cwafv1.Ingress{
				Name:         randomString(),
				Host:         randomString(),
				Port:         80,
				Path:         "",
				UpstreamHost: randomString(),
				UpstreamPort: 90,
			},
		},
	)
	_, err := svc.CreateIngress(context.Background(), req)
	assert.Nil(t, err)
}

func TestCreateIngressWithExistingApp(t *testing.T) {
	appSvc := NewApplicationService(applogger.NewLogger())
	app, err := appSvc.CreateApplication(
		context.Background(),
		connect.NewRequest(
			&cwafv1.CreateApplicationRequest{
				Name: randomString(),
			},
		),
	)
	assert.Nil(t, err)
	svc := NewIngressService(applogger.NewLogger())
	_, err = svc.CreateIngress(context.Background(),
		connect.NewRequest(
			&cwafv1.CreateIngressRequest{
				Ingress: &cwafv1.Ingress{
					Name:          randomString(),
					Host:          randomString(),
					Port:          80,
					Path:          "",
					UpstreamHost:  randomString(),
					UpstreamPort:  90,
					ApplicationId: int32(app.Msg.Id),
				},
			},
		),
	)

	assert.Nil(t, err)

}
