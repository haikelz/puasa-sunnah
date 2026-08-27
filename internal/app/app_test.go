package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"puasa-sunnah/internal/config"
)

func TestAPI(t *testing.T) {
	t.Parallel()

	server, err := New(config.Config{AppName: "test", Port: 8080})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("health", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/health", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
	})

	t.Run("classify", func(t *testing.T) {
		body := []byte(`{"date":"2026-06-25","hijri":{"year":1448,"month":1,"day":10}}`)
		request, err := http.NewRequest(http.MethodPost, "/api/en/fasting/classify", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusOK, payload)
		}

		var payload struct {
			Data struct {
				Verdict string `json:"verdict"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Data.Verdict != "recommended" {
			t.Fatalf("verdict = %q, want recommended", payload.Data.Verdict)
		}
	})

	t.Run("monthly schedule", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/api/en/fasting/schedules?period=month&date=2026-08-09", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusOK, payload)
		}
		var payload struct {
			Data struct {
				Calendar       string `json:"calendar"`
				Total          int    `json:"total"`
				CalendarSource struct {
					ID string `json:"id"`
				} `json:"calendar_source"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Data.Calendar != "id" || payload.Data.Total != 11 || payload.Data.CalendarSource.ID != "kemenag-ri-2026" {
			t.Fatalf("schedule calendar/total/source = %q/%d/%q, want id/11/kemenag-ri-2026", payload.Data.Calendar, payload.Data.Total, payload.Data.CalendarSource.ID)
		}
	})

	t.Run("Umm al-Qura schedule", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/api/id/fasting/schedules?period=year&date=2027-08-09&calendar=umm-al-qura", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusOK, payload)
		}

		var payload struct {
			Data struct {
				Calendar       string `json:"calendar"`
				Total          int    `json:"total"`
				CalendarSource struct {
					ID     string `json:"id"`
					Caveat string `json:"caveat"`
				} `json:"calendar_source"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Data.Calendar != "umm-al-qura" || payload.Data.Total == 0 || payload.Data.CalendarSource.ID != "umm-al-qura" {
			t.Fatalf("schedule calendar/total/source = %q/%d/%q", payload.Data.Calendar, payload.Data.Total, payload.Data.CalendarSource.ID)
		}
		if !strings.Contains(payload.Data.CalendarSource.Caveat, "Arab Saudi") {
			t.Errorf("Indonesian Umm al-Qura caveat = %q", payload.Data.CalendarSource.Caveat)
		}
	})

	t.Run("Indonesian schedule", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/api/id/fasting/schedules?period=week&date=2026-08-09", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusOK, payload)
		}

		var payload struct {
			Data struct {
				WeekStartsOn string `json:"week_starts_on"`
				Disclaimer   string `json:"disclaimer"`
				Dates        []struct {
					Weekday         string `json:"weekday"`
					Recommendations []struct {
						Name string `json:"name"`
					} `json:"recommendations"`
				} `json:"dates"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Data.WeekStartsOn != "Senin" || len(payload.Data.Dates) == 0 {
			t.Fatalf("Indonesian schedule start/dates = %q/%d, want Senin/non-empty", payload.Data.WeekStartsOn, len(payload.Data.Dates))
		}
		if len(payload.Data.Dates[0].Recommendations) == 0 {
			t.Fatal("Indonesian first schedule date has no recommendations")
		}
		if payload.Data.Dates[0].Weekday != "Senin" || payload.Data.Dates[0].Recommendations[0].Name != "Senin atau Kamis" {
			t.Errorf("Indonesian first date = %q/%q", payload.Data.Dates[0].Weekday, payload.Data.Dates[0].Recommendations[0].Name)
		}
		if !strings.Contains(payload.Data.Disclaimer, "Informasi ini hanya untuk perencanaan") {
			t.Errorf("Indonesian disclaimer = %q", payload.Data.Disclaimer)
		}
	})

	t.Run("Indonesian rules", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/api/id/fasting/rules", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var payload struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || len(payload.Data) == 0 {
			t.Fatalf("Indonesian rules status/count = %d/%d", response.StatusCode, len(payload.Data))
		}
		if payload.Data[0].Name != "Puasa Senin dan Kamis" {
			t.Fatalf("Indonesian rules status/first name = %d/%q", response.StatusCode, payload.Data[0].Name)
		}
	})

	t.Run("Indonesian validation errors", func(t *testing.T) {
		tests := []struct {
			path       string
			wantStatus int
			wantText   string
		}{
			{"/api/id/fasting/schedules?period=decade&date=2026-08-09", http.StatusBadRequest, "period harus bernilai"},
			{"/api/id/fasting/schedules?period=year&date=2026-08-09&calendar=other", http.StatusBadRequest, "calendar harus bernilai"},
			{"/api/id/fasting/schedules?period=year&date=2027-08-09", http.StatusUnprocessableEntity, "kalender hanya mencakup"},
		}
		for _, tt := range tests {
			request, err := http.NewRequest(http.MethodGet, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tt.wantStatus || !strings.Contains(string(body), tt.wantText) {
				t.Errorf("GET %s status/body = %d/%s", tt.path, response.StatusCode, body)
			}
		}
	})

	t.Run("unsupported and removed routes", func(t *testing.T) {
		for _, path := range []string{"/api/fr/fasting/rules", "/api/v1/fasting/rules"} {
			request, err := http.NewRequest(http.MethodGet, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != http.StatusNotFound {
				t.Errorf("GET %s status = %d, want %d", path, response.StatusCode, http.StatusNotFound)
			}
		}
	})

	t.Run("swagger specification", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
		var specification struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}
		if err := json.NewDecoder(response.Body).Decode(&specification); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/health", "/api/{language}/fasting/rules", "/api/{language}/fasting/schedules", "/api/{language}/fasting/classify"} {
			if _, ok := specification.Paths[path]; !ok {
				t.Errorf("Swagger specification does not contain %s", path)
			}
		}
	})

	t.Run("swagger UI", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Puasa Sunnah API Documentation") {
			t.Fatalf("Swagger UI status/title invalid: status=%d", response.StatusCode)
		}
	})
}
