package models

import "time"

//Create type
type DevEUI struct {
	DevEUI    string `json:"deveui,omitempty"`
	ShortCode string `json:"shortcode,omitempty"`
}

type LastDevEUI struct {
	ShortCode string `json:"shortcode,omitempty"`
}

type RegisteredDevEUIList struct {
	DevEUIs []string `json:"deveuis,omitempty"`
}

type ResponseObject struct {
	Status string `json:"status,omitempty"`
	Code   int    `json:"code,omitempty"`
	Error  error  `json:"error,omitempty"`
}

type ApiResponseCacheObject struct {
	Key      string
	Response string
	Timeout  time.Duration
}
