package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"puasa-sunnah/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Cache-Control values the fasting handler promises to the CDN.
const (
	wantCacheFixedDate = "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"
	wantCacheToday     = "public, max-age=300, s-maxage=300"
)

func TestAPI(t *testing.T) {
	t.Parallel()

	server, err := New(config.Config{AppName: "test", Port: 8080})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("filtered calendar export", func(t *testing.T) {
		for _, language := range []string{"en", "id"} {
			request, err := http.NewRequest(http.MethodGet, "/api/"+language+"/fasting/schedules?period=month&date=2026-08-09&types=white_days&format=ics", nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			unfolded := strings.ReplaceAll(string(body), "\r\n ", "")
			if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/calendar") || !strings.Contains(response.Header.Get("Content-Disposition"), ".ics") {
				t.Fatalf("export response: %d %v", response.StatusCode, response.Header)
			}
			if strings.Count(unfolded, "BEGIN:VEVENT") != 3 || !strings.Contains(unfolded, "DTSTART;VALUE=DATE:20260827\r\nDTEND;VALUE=DATE:20260828") || !strings.Contains(unfolded, "https://aceh.kemenag.go.id/") || !strings.Contains(unfolded, "Sidang Isbat") {
				t.Fatalf("unexpected calendar: %s", unfolded)
			}
			for _, line := range strings.Split(string(body), "\r\n") {
				if len(line) > 75 {
					t.Fatalf("line exceeds 75 bytes: %q", line)
				}
			}
		}
	})

	t.Run("export validation and empty calendar", func(t *testing.T) {
		for _, query := range []string{"types=unknown", "types=white_days,", "format=pdf"} {
			request, err := http.NewRequest(http.MethodGet, "/api/id/fasting/schedules?period=month&date=2026-08-09&"+query, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			_ = response.Body.Close()
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("%s status = %d", query, response.StatusCode)
			}
		}
		request, err := http.NewRequest(http.MethodGet, "/api/en/fasting/schedules?period=month&date=2026-08-09&types=ashura&format=ics", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 200 || !strings.HasSuffix(string(body), "END:VCALENDAR\r\n") || strings.Contains(string(body), "BEGIN:VEVENT") {
			t.Fatalf("empty calendar: %d %s", response.StatusCode, body)
		}
	})

	t.Run("input normalization", func(t *testing.T) {
		spaced := sendRequest(
			t,
			server,
			http.MethodGet,
			"/api/en/fasting/schedules?period=month&date=2026-08-09&types=white_days,%20arafah",
			"",
		)
		if spaced.status != http.StatusOK {
			t.Errorf("types with spaces status = %d, want %d", spaced.status, http.StatusOK)
		}

		form := sendRequestWithType(
			t,
			server,
			http.MethodPost,
			"/api/en/fasting/classify",
			"date=2026-06-25",
			"application/x-www-form-urlencoded",
		)
		if form.status != http.StatusBadRequest || !strings.Contains(form.body, "body must be valid JSON") {
			t.Errorf("form classify status/body = %d/%s", form.status, form.body)
		}
	})

	t.Run("cache headers", func(t *testing.T) {
		tests := []struct {
			path string
			want string
		}{
			{
				path: "/api/en/fasting/schedules?period=month&date=2026-08-09",
				want: wantCacheFixedDate,
			},
			{
				path: "/api/en/fasting/schedules?period=month&date=2026-08-09&format=ics",
				want: wantCacheFixedDate,
			},
			// No date means today; Umm al-Qura keeps this case inside coverage for decades.
			{
				path: "/api/en/fasting/schedules?period=week&calendar=umm-al-qura",
				want: wantCacheToday,
			},
			{
				path: "/api/id/fasting/rules",
				want: wantCacheFixedDate,
			},
			{
				path: "/api/en/fasting/schedules?period=decade&date=2026-08-09",
				want: cacheNoStore,
			},
			{
				path: "/health",
				want: cacheNoStore,
			},
		}

		for _, tt := range tests {
			got := sendRequest(t, server, http.MethodGet, tt.path, "")

			if got.cacheControl != tt.want {
				t.Errorf("GET %s Cache-Control = %q, want %q", tt.path, got.cacheControl, tt.want)
			}
		}
	})

	t.Run("day lookup", func(t *testing.T) {
		tests := []struct {
			method     string
			path       string
			body       string
			wantStatus int
			wantCache  string
			wantText   string
		}{
			{
				method:     http.MethodGet,
				path:       "/api/id/fasting/days/2027-03-10",
				wantStatus: http.StatusOK,
				wantCache:  wantCacheFixedDate,
				wantText:   `"verdict":"prohibited"`,
			},
			{
				method:     http.MethodGet,
				path:       "/api/en/fasting/days/today?calendar=umm-al-qura",
				wantStatus: http.StatusOK,
				wantCache:  wantCacheToday,
				wantText:   `"patterns":[]`,
			},
			{
				method:     http.MethodPost,
				path:       "/api/en/fasting/days/2027-08-06",
				body:       `{"fasting_dates":[],"observes_dawud":false}`,
				wantStatus: http.StatusOK,
				wantCache:  cacheNoStore,
				wantText:   `"status":"caution"`,
			},
			{
				method:     http.MethodPost,
				path:       "/api/id/fasting/days/2027-08-06",
				body:       `{"fasting_dates":["2027-08-05"]}`,
				wantStatus: http.StatusOK,
				wantCache:  cacheNoStore,
				wantText:   `"status":"paired"`,
			},
			{
				method:     http.MethodPost,
				path:       "/api/en/fasting/days/2027-08-06",
				body:       `{"fasted":["2027-08-05"]}`,
				wantStatus: http.StatusBadRequest,
				wantCache:  cacheNoStore,
				wantText:   "fasting_dates",
			},
			{
				method:     http.MethodPost,
				path:       "/api/en/fasting/days/2027-08-06",
				body:       `not json`,
				wantStatus: http.StatusBadRequest,
				wantCache:  cacheNoStore,
				wantText:   "body must be JSON",
			},
			{
				method:     http.MethodGet,
				path:       "/api/en/fasting/days/06-08-2027",
				wantStatus: http.StatusBadRequest,
				wantCache:  cacheNoStore,
				wantText:   "date must use YYYY-MM-DD",
			},
			{
				method:     http.MethodGet,
				path:       "/api/en/fasting/days/2028-01-03",
				wantStatus: http.StatusUnprocessableEntity,
				wantCache:  cacheNoStore,
				wantText:   "calendar=umm-al-qura",
			},
		}

		for _, tt := range tests {
			got := sendRequest(t, server, tt.method, tt.path, tt.body)

			isExpected := got.status == tt.wantStatus &&
				got.cacheControl == tt.wantCache &&
				strings.Contains(got.body, tt.wantText)
			if !isExpected {
				t.Errorf("%s %s = %d %q %s", tt.method, tt.path, got.status, got.cacheControl, got.body)
			}
		}
	})

	t.Run("rolling feed", func(t *testing.T) {
		feed := sendRequest(
			t,
			server,
			http.MethodGet,
			"/api/id/fasting/schedules?period=range&days=60&calendar=umm-al-qura&format=ics&reminder=20:00",
			"",
		)

		unfolded := strings.ReplaceAll(feed.body, "\r\n ", "")
		isExpected := feed.status == http.StatusOK &&
			feed.cacheControl == wantCacheToday &&
			strings.Contains(unfolded, "TRIGGER:-PT4H") &&
			strings.Contains(unfolded, "X-WR-CALNAME:Puasa Sunnah")
		if !isExpected {
			t.Fatalf("rolling feed = %d %q %s", feed.status, feed.cacheControl, unfolded)
		}

		invalidQueries := []string{
			"period=range&days=ten",
			"period=range&days=5&reminder=20:00",
			"period=range&days=5&format=ics&reminder=8pm",
		}
		for _, query := range invalidQueries {
			got := sendRequest(t, server, http.MethodGet, "/api/en/fasting/schedules?"+query, "")

			if got.status != http.StatusBadRequest {
				t.Errorf("%s status = %d, want 400", query, got.status)
			}
		}
	})

	t.Run("health", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/health", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := server.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
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
		defer func() { _ = response.Body.Close() }()
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
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusOK, payload)
		}
		var payload struct {
			Data struct {
				Calendar       string `json:"calendar"`
				Total          int    `json:"total"`
				CalendarSource struct {
					ID           string `json:"id"`
					Publications []struct {
						ID string `json:"id"`
					} `json:"publications"`
				} `json:"calendar_source"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		source := payload.Data.CalendarSource
		isExpected := payload.Data.Calendar == "id" &&
			payload.Data.Total == 11 &&
			source.ID == "kemenag-ri" &&
			len(source.Publications) == 1 &&
			source.Publications[0].ID == "kemenag-ri-2026"
		if !isExpected {
			t.Fatalf("schedule calendar/total/source = %q/%d/%+v, want id/11/kemenag-ri-2026", payload.Data.Calendar, payload.Data.Total, source)
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
		defer func() { _ = response.Body.Close() }()
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
		defer func() { _ = response.Body.Close() }()
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
		defer func() { _ = response.Body.Close() }()
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
			{
				path:       "/api/id/fasting/schedules?period=decade&date=2026-08-09",
				wantStatus: http.StatusBadRequest,
				wantText:   "period harus bernilai",
			},
			{
				path:       "/api/id/fasting/schedules?period=year&date=2026-08-09&calendar=other",
				wantStatus: http.StatusBadRequest,
				wantText:   "calendar harus bernilai",
			},
			{
				path:       "/api/id/fasting/schedules?period=year&date=2028-08-09",
				wantStatus: http.StatusUnprocessableEntity,
				wantText:   "kalender hanya mencakup 2025-12-28 sampai 2027-12-31; untuk tanggal lain, calendar=umm-al-qura",
			},
			{
				path:       "/api/id/fasting/schedules?period=year&date=2027-08-09",
				wantStatus: http.StatusOK,
				wantText:   "kemenag-ri-2027",
			},
		}

		for _, tt := range tests {
			got := sendRequest(t, server, http.MethodGet, tt.path, "")

			if got.status != tt.wantStatus || !strings.Contains(got.body, tt.wantText) {
				t.Errorf("GET %s status/body = %d/%s", tt.path, got.status, got.body)
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
			_ = response.Body.Close()
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
		defer func() { _ = response.Body.Close() }()
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
		defer func() { _ = response.Body.Close() }()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Puasa Sunnah API Documentation") {
			t.Fatalf("Swagger UI status/title invalid: status=%d", response.StatusCode)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()

	server := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})
	server.Use(
		accessLog,
		recover.New(),
		withRequestTimeout,
	)
	server.Get("/api/:language/internal", func(*fiber.Ctx) error {
		return errors.New("secret detail")
	})
	server.Get("/api/:language/timeout", func(c *fiber.Ctx) error {
		if _, ok := c.UserContext().Deadline(); !ok {
			return errors.New("request context has no deadline")
		}

		return context.DeadlineExceeded
	})
	server.Get("/api/:language/panic", func(*fiber.Ctx) error {
		panic("boom")
	})

	tests := []struct {
		path       string
		wantStatus int
		wantText   string
	}{
		{
			path:       "/api/en/internal",
			wantStatus: http.StatusInternalServerError,
			wantText:   "internal server error",
		},
		{
			path:       "/api/id/internal",
			wantStatus: http.StatusInternalServerError,
			wantText:   "kesalahan internal server",
		},
		{
			path:       "/api/en/panic",
			wantStatus: http.StatusInternalServerError,
			wantText:   "internal server error",
		},
		{
			path:       "/api/id/timeout",
			wantStatus: http.StatusServiceUnavailable,
			wantText:   "permintaan melebihi batas waktu",
		},
	}

	for _, tt := range tests {
		got := sendRequest(t, server, http.MethodGet, tt.path, "")

		isExpected := got.status == tt.wantStatus &&
			strings.Contains(got.body, tt.wantText) &&
			!strings.Contains(got.body, "secret")
		if !isExpected {
			t.Errorf("GET %s status/body = %d/%s", tt.path, got.status, got.body)
		}
	}
}

func TestOversizedBodyKeepsServerAlive(t *testing.T) {
	t.Parallel()

	server, err := New(config.Config{AppName: "test"})
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		// Listener returns once Shutdown runs in Cleanup; that error is expected.
		_ = server.Listener(listener)
	}()
	t.Cleanup(func() {
		_ = server.Shutdown()
	})
	baseURL := "http://" + listener.Addr().String()

	body := strings.NewReader(`{"date":"` + strings.Repeat("a", bodyLimit) + `"}`)
	response, err := http.Post(baseURL+"/api/id/fasting/classify", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body status = %d, want %d", response.StatusCode, http.StatusRequestEntityTooLarge)
	}

	response, err = http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("server died after oversized body: %v", err)
	}
	_ = response.Body.Close()
}

type testResponse struct {
	status       int
	cacheControl string
	body         string
}

// sendRequest sends a JSON request through server.Test and reads the whole response.
func sendRequest(
	t *testing.T,
	server *fiber.App,
	method string,
	path string,
	body string,
) testResponse {
	t.Helper()

	return sendRequestWithType(t, server, method, path, body, "application/json")
}

func sendRequestWithType(
	t *testing.T,
	server *fiber.App,
	method string,
	path string,
	body string,
	contentType string,
) testResponse {
	t.Helper()

	request, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", contentType)

	response, err := server.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	return testResponse{
		status:       response.StatusCode,
		cacheControl: response.Header.Get("Cache-Control"),
		body:         string(content),
	}
}
