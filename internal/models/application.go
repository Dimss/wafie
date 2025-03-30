package models

import (
	"connectrpc.com/connect"
	"errors"
	"fmt"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"gorm.io/gorm"
	"time"
)

type Application struct {
	ID          uint         `gorm:"primaryKey"`
	Name        string       `gorm:"uniqueIndex:idx_name_namespace"`
	Namespace   string       `gorm:"uniqueIndex:idx_name_namespace"`
	Protections []Protection `gorm:"foreignKey:ApplicationID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

//func (a *Application) ToCwafV1App() *v1.Application {
//	protected := v1.AppProtected_NO
//	if a.Protected {
//		protected = v1.AppProtected_YES
//	}
//	return &v1.Application{
//		Id:        uint32(a.ID),
//		Name:      a.Name,
//		Namespace: a.Namespace,
//		Protected: protected,
//	}
//}

func GetApplication(req *v1.GetApplicationRequest) (*Application, error) {
	app := &Application{
		ID: uint(req.GetId()),
	}
	err := db().Preload("Protections.WAFConfig").First(&app, req.GetId()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("application not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return app, nil
}

func CreateApplication(req *v1.CreateApplicationRequest) (*Application, error) {
	app, err := createFromProtoToApplication(req)
	if err != nil {
		return nil, err
	}

	if err := db().Create(app).Error; err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	return app, nil
}

//func ListApplications(options *v1.ListApplicationsOptions) ([]*Application, error) {
//	var apps []*Application
//	whereMap := map[string]interface{}{}
//	if options.Protected == v1.AppProtected_NO {
//		whereMap["protected"] = false
//	}
//	if options.Protected == v1.AppProtected_YES {
//		whereMap["protected"] = true
//	}
//	res := db().Where(whereMap).Find(&apps)
//	if res.Error != nil {
//		return nil, connect.NewError(connect.CodeUnknown, res.Error)
//	}
//	return apps, nil
//}
//
//func UpdateApplication(cwafv1app *v1.Application) error {
//	appToUpdate := &Application{}
//	if res := db().First(appToUpdate, uint(cwafv1app.Id)); res.Error != nil {
//		return connect.NewError(connect.CodeUnknown, res.Error)
//	}
//	if cwafv1app.Name != "" {
//		appToUpdate.Name = cwafv1app.Name
//	}
//	if cwafv1app.Namespace != "" {
//		appToUpdate.Namespace = cwafv1app.Namespace
//	}
//	if cwafv1app.Protected != v1.AppProtected_UNSPECIFIED {
//		if cwafv1app.Protected == v1.AppProtected_YES {
//			appToUpdate.Protected = true
//		}
//		if cwafv1app.Protected == v1.AppProtected_NO {
//			appToUpdate.Protected = false
//		}
//	}
//	res := db().Save(appToUpdate)
//	if res.Error != nil {
//		return connect.NewError(connect.CodeUnknown, res.Error)
//	}
//	return nil
//}
