package model

import (
	"connectrpc.com/connect"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"gorm.io/gorm"
	"time"
)

type Application struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex:idx_app_name_namespace"`
	Namespace   string `gorm:"uniqueIndex:idx_app_name_namespace"`
	Description string `gorm:"type:text"`
	Protected   bool   `gorm:"default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewApplicationFromRequest(req *v1.CreateApplicationRequest) *Application {
	return &Application{
		Name:        req.GetName(),
		Namespace:   req.GetNamespace(),
		Description: req.GetDescription(),
		Protected:   req.GetProtected(),
	}
}

func GetApplicationByNameOrId(req *v1.GetApplicationRequest, db *gorm.DB) (*Application, error) {

	app := &Application{
		ID:   uint(req.GetId()),
		Name: req.GetName(),
	}
	res := db.Where(app).First(app)
	if res.RowsAffected == 0 {
		return app, connect.NewError(connect.CodeNotFound, res.Error)
	}
	if res.Error != nil {
		return app, connect.NewError(connect.CodeUnknown, res.Error)
	}
	return app, nil

}
