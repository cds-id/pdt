package database

import (
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Commit{},
		&models.CommitCardLink{},
		&models.Sprint{},
		&models.JiraCard{},
		&models.ReportTemplate{},
		&models.Report{},
	)
}
