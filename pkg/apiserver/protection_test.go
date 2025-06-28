package apiserver

import (
	"connectrpc.com/connect"
	"context"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func createProtectionDependencies(t *testing.T) (appId uint32) {
	// create new application
	appSvc := NewApplicationService(applogger.NewLogger())
	app, err := appSvc.CreateApplication(
		context.Background(),
		connect.NewRequest(
			&wafiev1.CreateApplicationRequest{
				Name: randomString(),
			},
		),
	)
	assert.Nil(t, err)
	// create new ingress
	ingSvc := NewIngressService(applogger.NewLogger())
	_, err = ingSvc.CreateIngress(context.Background(),
		connect.NewRequest(
			&wafiev1.CreateIngressRequest{
				Ingress: &wafiev1.Ingress{
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
	_ = &wafiev1.CreateProtectionRequest{
		ApplicationId:  app.Msg.Id,
		ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
		DesiredState: &wafiev1.ProtectionDesiredState{
			ModeSec: &wafiev1.ModSec{
				ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
				ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
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
		connect.NewRequest(&wafiev1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Empty(t, vh.Msg.VirtualHosts[0].Spec)
}

func TestEnabledProtection(t *testing.T) {
	appId := createProtectionDependencies(t)
	protectionSvc := NewProtectionService(applogger.NewLogger())
	_, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&wafiev1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.NotEmpty(t, vh.Msg.VirtualHosts[0].Spec)
}

func TestProtectionTestModeOn(t *testing.T) {
	appId := createProtectionDependencies(t)
	//create new protection
	_ = &wafiev1.CreateProtectionRequest{
		ApplicationId:  appId,
		ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
		DesiredState: &wafiev1.ProtectionDesiredState{
			ModeSec: &wafiev1.ModSec{
				ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
				ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
			},
		},
	}
	protectionSvc := NewProtectionService(applogger.NewLogger())
	_, err := protectionSvc.CreateProtection(
		context.Background(),
		connect.NewRequest(&wafiev1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_ON,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
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
		connect.NewRequest(&wafiev1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_ON,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	virtualHostSvc := NewVirtualHostService(applogger.NewLogger())
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
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
		connect.NewRequest(&wafiev1.CreateProtectionRequest{
			ApplicationId:  appId,
			ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_ON,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	vh, err := virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Empty(t, vh.Msg.VirtualHosts[0].Spec)
	modeOn := wafiev1.ProtectionMode_PROTECTION_MODE_ON
	// update protection
	_, err = protectionSvc.PutProtection(
		context.Background(),
		connect.NewRequest(&wafiev1.PutProtectionRequest{
			Id:             protection.Msg.Protection.Id,
			ProtectionMode: &modeOn,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_ON,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	assert.Nil(t, err)
	// get rendered virtual host
	vh, err = virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.Contains(t, vh.Msg.VirtualHosts[0].Spec, "modsecurity on;")
	// update protection
	_, err = protectionSvc.PutProtection(
		context.Background(),
		connect.NewRequest(&wafiev1.PutProtectionRequest{
			Id:             protection.Msg.Protection.Id,
			ProtectionMode: &modeOn,
			DesiredState: &wafiev1.ProtectionDesiredState{
				ModeSec: &wafiev1.ModSec{
					ProtectionMode: wafiev1.ProtectionMode_PROTECTION_MODE_OFF,
					ParanoiaLevel:  wafiev1.ParanoiaLevel_PARANOIA_LEVEL_4,
				},
			},
		}),
	)
	assert.Nil(t, err)
	// get rendered virtual host
	vh, err = virtualHostSvc.ListVirtualHosts(
		context.Background(),
		connect.NewRequest(&wafiev1.ListVirtualHostsRequest{}))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(vh.Msg.VirtualHosts))
	assert.NotContains(t, vh.Msg.VirtualHosts[0].Spec, "modsecurity on;")

}
