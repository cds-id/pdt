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
	if err := db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Commit{},
		&models.CommitCardLink{},
		&models.JiraWorkspaceConfig{},
		&models.Sprint{},
		&models.JiraCard{},
		&models.ReportTemplate{},
		&models.Report{},
		&models.Conversation{},
		&models.ChatMessage{},
		&models.AIUsage{},
		&models.JiraComment{},
		&models.WaNumber{},
		&models.WaListener{},
		&models.WaMessage{},
		&models.WaMedia{},
		&models.WaOutbox{},
		&models.TelegramConfig{},
		&models.TelegramWhitelist{},
		&models.AgentSchedule{},
		&models.AgentScheduleRun{},
		&models.AgentScheduleRunStep{},
		&models.ComposioConfig{},
		&models.ComposioConnection{},
		&models.ExecutiveReport{},
	); err != nil {
		return err
	}

	// Backfill updated_at for jira_cards rows that were created before the
	// column was added (those will have updated_at = zero/NULL after AutoMigrate).
	db.Exec("UPDATE jira_cards SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = '0001-01-01 00:00:00'")

	// Migrate existing single-workspace users to JiraWorkspaceConfig
	migrateJiraWorkspaces(db)

	return nil
}

// migrateJiraWorkspaces creates JiraWorkspaceConfig entries for users
// that have Jira configured on the User model but no workspace entries yet.
func migrateJiraWorkspaces(db *gorm.DB) {
	var users []models.User
	db.Where("jira_workspace != '' AND jira_token != ''").Find(&users)

	for _, user := range users {
		var count int64
		db.Model(&models.JiraWorkspaceConfig{}).Where("user_id = ?", user.ID).Count(&count)
		if count > 0 {
			continue // already migrated
		}

		ws := models.JiraWorkspaceConfig{
			UserID:      user.ID,
			Workspace:   user.JiraWorkspace,
			Name:        user.JiraWorkspace,
			ProjectKeys: user.JiraProjectKeys,
			IsActive:    true,
		}
		db.Create(&ws)

		// Backfill workspace_id on existing records
		db.Model(&models.Sprint{}).Where("user_id = ? AND workspace_id IS NULL", user.ID).Update("workspace_id", ws.ID)
		db.Model(&models.JiraCard{}).Where("user_id = ? AND workspace_id IS NULL", user.ID).Update("workspace_id", ws.ID)
		db.Model(&models.JiraComment{}).Where("user_id = ? AND workspace_id IS NULL", user.ID).Update("workspace_id", ws.ID)
	}
}
