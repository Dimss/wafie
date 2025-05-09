package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func createProtectionDependencies(t *testing.T) (appId uint32) {
	// create new application
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
	// create new ingress
	ingSvc := NewIngressService(applogger.NewLogger())
	_, err = ingSvc.CreateIngress(context.Background(),
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
	//create new protection
	_ = &cwafv1.CreateProtectionRequest{
		ApplicationId:  app.Msg.Id,
		ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
		DesiredState: &cwafv1.ProtectionDesiredState{
			ModeSec: &cwafv1.ModSec{
				ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
				ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
			},
		},
	}
	return app.Msg.Id
}

func TestDisabledProtection(t *testing.T) {
	appId := createProtectionDependencies(t)
	//create new protection
	protectionSvc := NewProtectionService(applogger.NewLogger())
	_, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Empty(t, vh.Msg.VirtualHosts[0].Spec)
}

func TestEnabledProtection(t *testing.T) {
	appId := createProtectionDependencies(t)
	protectionSvc := NewProtectionService(applogger.NewLogger())
	_, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.NotEmpty(t, vh.Msg.VirtualHosts[0].Spec)
}

func TestProtectionTestModeOn(t *testing.T) {
	appId := createProtectionDependencies(t)
	//create new protection
	_ = &cwafv1.CreateProtectionRequest{
		ApplicationId:  appId,
		ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
		DesiredState: &cwafv1.ProtectionDesiredState{
			ModeSec: &cwafv1.ModSec{
				ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
				ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
			},
		},
	}
	protectionSvc := NewProtectionService(applogger.NewLogger())
	_, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Contains(t, vh.Msg.VirtualHosts[0].Spec, "modsecurity on;")
}

func TestProtectionTestModeOff(t *testing.T) {
	appId := createProtectionDependencies(t)
	//create new protection
	protectionSvc := NewProtectionService(applogger.NewLogger())
	_, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.NotContains(t, vh.Msg.VirtualHosts[0].Spec, "modsecurity on;")

}

func TestUpdateProtection(t *testing.T) {
	appId := createProtectionDependencies(t)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	//create new protection
	protectionSvc := NewProtectionService(applogger.NewLogger())
	protection, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Empty(t, vh.Msg.VirtualHosts[0].Spec)
	// update protection
	_, err = protectionSvc.PutProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.PutProtectionRequest{
			Id:             protection.Msg.Protection.Id,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	assert.Nil(t, err)
	// get rendered virtual host
	vh, err = virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Contains(t, vh.Msg.VirtualHosts[0].Spec, "modsecurity on;")

	// update protection
	_, err = protectionSvc.PutProtection(
		context.Background(),
		connect.NewRequest(&cwafv1.PutProtectionRequest{
			Id:             protection.Msg.Protection.Id,
			ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &cwafv1.ProtectionDesiredState{
				ModeSec: &cwafv1.ModSec{
					ProtectionMode: cwafv1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  cwafv1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	assert.Nil(t, err)
	// get rendered virtual host
	vh, err = virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.NotContains(t, vh.Msg.VirtualHosts[0].Spec, "modsecurity on;")

}
