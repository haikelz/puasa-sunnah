package http

import "puasa-sunnah/internal/domains/fasting"

type RulesResponse struct {
	Data []fasting.Rule `json:"data"`
}

type ScheduleResponse struct {
	Data fasting.Schedule `json:"data"`
}

type ClassificationResponse struct {
	Data fasting.Classification `json:"data"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"invalid request"`
}
