package models

import (
	"time"

	"github.com/Dimss/wafie/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EndpointSlice struct {
	ID           uint `gorm:"primary_key"`
	IP           string
	PodName      string
	Namespace    string
	NodeName     string
	PodUID       string
	Ports        string `gorm:"type:text"`
	UpstreamHost string // Foreign key field referencing Ingress.UpstreamHost
	//Ingress      Ingress `gorm:"foreignKey:UpstreamHost;references:UpstreamHost"`
	//Posts        []Post `gorm:"foreignKey:UserNumber;references:MemberNumber"` // Define the one-to-many relationship
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EndpointSliceSvc struct {
	db      *gorm.DB
	logger  *zap.Logger
	Ingress Ingress
}

func NewEndpointSlice(tx *gorm.DB, logger *zap.Logger) *EndpointSliceSvc {
	modelSvc := &EndpointSliceSvc{db: tx, logger: logger}

	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}

	return modelSvc
}
