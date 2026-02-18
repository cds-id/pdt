package models

import "time"

type Commit struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	RepoID      uint       `gorm:"index;not null" json:"repo_id"`
	SHA         string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"sha"`
	Message     string     `gorm:"type:text" json:"message"`
	Author      string     `gorm:"type:varchar(255)" json:"author"`
	AuthorEmail string     `gorm:"type:varchar(255)" json:"author_email"`
	Branch      string     `gorm:"type:varchar(255);index" json:"branch"`
	Date        time.Time  `json:"date"`
	JiraCardKey string     `gorm:"type:varchar(50);index" json:"jira_card_key"`
	HasLink     bool       `gorm:"default:false" json:"has_link"`
	CreatedAt   time.Time  `json:"created_at"`
	Repository  Repository `gorm:"foreignKey:RepoID" json:"-"`
}

type CommitCardLink struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CommitID    uint      `gorm:"index;not null" json:"commit_id"`
	JiraCardKey string    `gorm:"type:varchar(50);not null" json:"jira_card_key"`
	LinkedAt    time.Time `json:"linked_at"`
	Commit      Commit    `gorm:"foreignKey:CommitID" json:"-"`
}
