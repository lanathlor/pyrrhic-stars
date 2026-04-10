package persistence

import "time"

// Player represents a persistent player account identified by a client-generated UUID.
type Player struct {
	ID        string `gorm:"primaryKey;size:36"`
	Username  string `gorm:"size:20"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Character represents a named character belonging to a player.
// A player may have multiple characters of the same class.
type Character struct {
	ID        uint      `gorm:"primaryKey"`
	PlayerID  string    `gorm:"size:36;index"`
	ClassName string    `gorm:"size:20"`
	Name      string    `gorm:"size:20;uniqueIndex"`
	PosX      float64
	PosY      float64
	PosZ      float64
	RotY      float64
	CreatedAt time.Time
	UpdatedAt time.Time
}
