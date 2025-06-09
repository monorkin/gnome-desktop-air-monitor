package api

import (
	"time"
)

type Measurement struct {
	Timestamp   time.Time `json:"timestamp,omitempty"`
	Score       int       `json:"score,omitempty"`
	DewPoint    float64   `json:"dew_point,omitempty"`
	Temperature float64   `json:"temp,omitempty"`
	Humidity    float64   `json:"humid,omitempty"`
	CO2         int       `json:"co2,omitempty"`
	VOC         int       `json:"voc,omitempty"`
	PM25        int       `json:"pm25,omitempty"`
}
