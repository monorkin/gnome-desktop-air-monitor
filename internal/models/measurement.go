package models

import (
	"time"

	"gorm.io/gorm"
)

type Measurement struct {
	gorm.Model
	DeviceID    uint
	Timestamp   time.Time `gorm:"index"`
	Temperature float64
	Humidity    float64
	CO2         float64
	VOC         float64
	PM25        float64
	Score       float64
}
