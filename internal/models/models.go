package models

import "time"

// User represents an application user.
type User struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    string    `gorm:"uniqueIndex;size:64"`
	FirstSeen time.Time `gorm:"index"`
	Platform  string    `gorm:"size:32;index"`
	Region    string    `gorm:"size:64;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserEvent represents a single user event reported from the app.
type UserEvent struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    string    `gorm:"index;size:64"`
	EventType string    `gorm:"size:32;index"`
	Platform  string    `gorm:"size:32;index"`
	Region    string    `gorm:"size:64;index"`
	EventTime time.Time `gorm:"index"`
	CreatedAt time.Time
}


