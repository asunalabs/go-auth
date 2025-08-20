package utils

type Response struct {
	Success bool   `json:"success"`
	Code    uint   `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
