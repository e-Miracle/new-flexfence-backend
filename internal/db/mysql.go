package db

import (
	"fmt"

	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectMySQL(host, port, user, password, database string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC", user, password, host, port, database)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&mysqlstore.OrganizationModel{},
		&mysqlstore.BusinessUserModel{},
		&mysqlstore.UserModel{},
		&mysqlstore.EventModel{},
		&mysqlstore.FenceModel{},
		&mysqlstore.EventJoinModel{},
		&mysqlstore.AttendanceModel{},
		&mysqlstore.UserActivitySessionModel{},
		&mysqlstore.FenceCaptureSessionModel{},
		&mysqlstore.ConsentTemplateModel{},
		&mysqlstore.OrganizationConsentFieldModel{},
		&mysqlstore.BusinessOTPChallengeModel{},
		&mysqlstore.UserOTPChallengeModel{},
		&mysqlstore.GeofenceAlertModel{},
	)
}
