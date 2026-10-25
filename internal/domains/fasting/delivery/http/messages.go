package http

import (
	"errors"

	domain "puasa-sunnah/internal/domains/fasting"

	"github.com/gofiber/fiber/v2"
)

// usecaseError maps use-case errors to client responses. Known request errors
// get a localized message; anything else goes to the app error handler, which
// logs it and answers with a fixed message.
func usecaseError(err error, language domain.Language) error {
	var requestError *domain.RequestError
	if !errors.As(err, &requestError) {
		return err
	}

	status := fiber.StatusBadRequest
	if errors.Is(err, domain.ErrCalendarUnavailable) {
		status = fiber.StatusUnprocessableEntity
	}

	return fiber.NewError(status, requestErrorMessage(requestError, language))
}

func requestErrorMessage(err *domain.RequestError, language domain.Language) string {
	switch err.Code {
	case domain.CodeDateFormat:
		indonesianField := err.Field
		if err.Field == "date" {
			indonesianField = "tanggal"
		}

		return localized(
			language,
			err.Field+" must use YYYY-MM-DD",
			indonesianField+" harus menggunakan format YYYY-MM-DD",
		)
	case domain.CodeHijriRange:
		return localized(
			language,
			"hijri year, month, or day is outside its supported range",
			"tahun, bulan, atau tanggal Hijriah berada di luar rentang yang didukung",
		)
	case domain.CodeUnsupportedType:
		return localized(
			language,
			"types contains an unsupported fasting type",
			"types berisi jenis puasa yang tidak didukung",
		)
	case domain.CodeUnsupportedCalendar:
		return localized(
			language,
			"calendar must be id or umm-al-qura",
			"calendar harus bernilai id atau umm-al-qura",
		)
	case domain.CodeUnsupportedPeriod:
		return localized(
			language,
			"period must be week, month, year, or range",
			"period harus bernilai week, month, year, atau range",
		)
	case domain.CodeRangeOnlyInput:
		return localized(
			language,
			"start, end, and days apply only to period=range",
			"start, end, dan days hanya berlaku untuk period=range",
		)
	case domain.CodeRangeWithDate:
		return localized(
			language,
			"period=range uses start instead of date",
			"period=range menggunakan start, bukan date",
		)
	case domain.CodeRangeEndAndDays:
		return localized(
			language,
			"use end or days, not both",
			"gunakan end atau days, jangan keduanya",
		)
	case domain.CodeRangeDays:
		return localized(
			language,
			"days must be from 1 to 366",
			"days harus bernilai 1 sampai 366",
		)
	case domain.CodeRangeNeedsEnd:
		return localized(
			language,
			"period=range needs end or days",
			"period=range memerlukan end atau days",
		)
	case domain.CodeRangeEndBeforeStart:
		return localized(
			language,
			"end must not precede start",
			"end tidak boleh sebelum start",
		)
	case domain.CodeRangeTooLong:
		return localized(
			language,
			"a range covers at most 366 days",
			"rentang paling panjang 366 hari",
		)
	case domain.CodeHistoryTooLong:
		return localized(
			language,
			"fasting_dates holds at most 366 dates",
			"fasting_dates paling banyak berisi 366 tanggal",
		)
	case domain.CodeOutsideCoverage:
		return coverageMessage(err, language)
	default:
		return localized(language, "invalid request", "permintaan tidak valid")
	}
}

// coverageMessage names calendar=umm-al-qura as an alternative for Kemenag
// gaps without switching to it: the caller must choose.
func coverageMessage(err *domain.RequestError, language domain.Language) string {
	message := localized(
		language,
		"calendar covers "+err.CoverageStart+" through "+err.CoverageEnd,
		"kalender hanya mencakup "+err.CoverageStart+" sampai "+err.CoverageEnd,
	)
	if err.Calendar == domain.CalendarIndonesia {
		message += localized(
			language,
			"; for other dates, calendar=umm-al-qura offers Saudi planning dates that can differ from Kemenag",
			"; untuk tanggal lain, calendar=umm-al-qura menyediakan tanggal perencanaan Arab Saudi yang dapat berbeda dari Kemenag",
		)
	}

	return message
}

func localized(
	language domain.Language,
	english string,
	indonesian string,
) string {
	if language == domain.LanguageIndonesian {
		return indonesian
	}

	return english
}
