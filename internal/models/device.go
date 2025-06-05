package models

import (
	"time"

	"gorm.io/gorm"
)

type Device struct {
	gorm.Model
	Name         string `gorm:"uniqueIndex"`
	IPAddress    string
	DeviceType   string
	SerialNumber string `gorm:"uniqueIndex"`
	LastSeen     time.Time
	Measurements []Measurement
}
