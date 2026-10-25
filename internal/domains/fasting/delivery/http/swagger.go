package http

import domain "puasa-sunnah/internal/domains/fasting"

type RulesResponse struct {
	Data []domain.Rule `json:"data"`
}

type ScheduleResponse struct {
	Data domain.Schedule `json:"data"`
}

type ClassificationResponse struct {
	Data domain.Classification `json:"data"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"invalid request"`
}
