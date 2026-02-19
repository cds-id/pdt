package models

import "time"

type User struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Email         string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash  string    `gorm:"-" json:"-"`
	Password      string    `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	GithubToken   string    `gorm:"type:text" json:"-"`
	GitlabToken   string    `gorm:"type:text" json:"-"`
	GitlabURL     string    `gorm:"type:varchar(500)" json:"gitlab_url"`
	JiraEmail     string    `gorm:"type:varchar(255)" json:"jira_email"`
	JiraToken     string    `gorm:"type:text" json:"-"`
	JiraWorkspace string    `gorm:"type:varchar(255)" json:"jira_workspace"`
	JiraUsername    string    `gorm:"type:varchar(255)" json:"jira_username"`
	JiraProjectKeys string    `gorm:"type:varchar(500)" json:"jira_project_keys"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
