package http

import (
	"errors"
	"strings"
	"testing"

	domain "puasa-sunnah/internal/domains/fasting"

	"github.com/gofiber/fiber/v2"
)

func TestRequestErrorMessagesInBothLanguages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err            *domain.RequestError
		wantEnglish    string
		wantIndonesian string
	}{
		{
			err: &domain.RequestError{
				Code:  domain.CodeDateFormat,
				Field: "date",
			},
			wantEnglish:    "date must use YYYY-MM-DD",
			wantIndonesian: "tanggal harus menggunakan format YYYY-MM-DD",
		},
		{
			err: &domain.RequestError{
				Code:  domain.CodeDateFormat,
				Field: "start",
			},
			wantEnglish:    "start must use YYYY-MM-DD",
			wantIndonesian: "start harus menggunakan format YYYY-MM-DD",
		},
		{
			err:            &domain.RequestError{Code: domain.CodeHijriRange},
			wantEnglish:    "hijri year, month, or day",
			wantIndonesian: "tahun, bulan, atau tanggal Hijriah",
		},
		{
			err:            &domain.RequestError{Code: domain.CodeUnsupportedType},
			wantEnglish:    "unsupported fasting type",
			wantIndonesian: "jenis puasa yang tidak didukung",
		},
		{
			err:            &domain.RequestError{Code: domain.CodeUnsupportedCalendar},
			wantEnglish:    "calendar must be id or umm-al-qura",
			wantIndonesian: "calendar harus bernilai id atau umm-al-qura",
		},
		{
			err:            &domain.RequestError{Code: domain.CodeUnsupportedPeriod},
			wantEnglish:    "period must be week, month, or year",
			wantIndonesian: "period harus bernilai week, month, atau year",
		},
		{
			err: &domain.RequestError{
				Code:          domain.CodeOutsideCoverage,
				Calendar:      domain.CalendarIndonesia,
				CoverageStart: "2025-12-28",
				CoverageEnd:   "2027-12-31",
			},
			wantEnglish:    "calendar covers 2025-12-28 through 2027-12-31; for other dates, calendar=umm-al-qura",
			wantIndonesian: "kalender hanya mencakup 2025-12-28 sampai 2027-12-31; untuk tanggal lain, calendar=umm-al-qura",
		},
	}

	for _, tt := range tests {
		english := requestErrorMessage(tt.err, domain.LanguageEnglish)
		indonesian := requestErrorMessage(tt.err, domain.LanguageIndonesian)

		if !strings.Contains(english, tt.wantEnglish) {
			t.Errorf("%s English = %q, want %q", tt.err.Code, english, tt.wantEnglish)
		}
		if !strings.Contains(indonesian, tt.wantIndonesian) {
			t.Errorf("%s Indonesian = %q, want %q", tt.err.Code, indonesian, tt.wantIndonesian)
		}
	}
}

func TestUsecaseErrorStatus(t *testing.T) {
	t.Parallel()

	ummAlQuraGap := &domain.RequestError{
		Kind:          domain.ErrCalendarUnavailable,
		Code:          domain.CodeOutsideCoverage,
		Calendar:      domain.CalendarUmmAlQura,
		CoverageStart: "1937-03-14",
		CoverageEnd:   "2077-11-16",
	}
	var fiberError *fiber.Error
	if err := usecaseError(ummAlQuraGap, domain.LanguageEnglish); !errors.As(err, &fiberError) {
		t.Fatalf("usecaseError() = %v, want a fiber error", err)
	}

	isExpected := fiberError.Code == fiber.StatusUnprocessableEntity &&
		!strings.Contains(fiberError.Message, "umm-al-qura")
	if !isExpected {
		t.Errorf("Umm al-Qura gap = %d %q, want 422 without a calendar hint", fiberError.Code, fiberError.Message)
	}

	invalid := &domain.RequestError{
		Kind:  domain.ErrInvalidRequest,
		Code:  domain.CodeUnsupportedCalendar,
		Field: "calendar",
	}
	if err := usecaseError(invalid, domain.LanguageEnglish); !errors.As(err, &fiberError) || fiberError.Code != fiber.StatusBadRequest {
		t.Errorf("invalid request = %v, want 400", err)
	}

	internal := errors.New("database exploded")
	if err := usecaseError(internal, domain.LanguageEnglish); err != internal {
		t.Errorf("unclassified error = %v, want it passed to the app error handler", err)
	}
}
