package usecase

import (
	"context"
	"errors"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

var (
	errIndonesiaCalendarRequired = errors.New("indonesia calendar is required")
	errUmmAlQuraCalendarRequired = errors.New("umm al-Qura calendar is required")
)

// FastingUsecase classifies dates and builds schedules from the calendar the
// caller selects.
type FastingUsecase struct {
	indonesiaCalendar domain.CalendarRepository
	ummAlQuraCalendar domain.CalendarRepository
}

type parsedRequest struct {
	date              time.Time
	hijri             domain.HijriDate
	isPilgrimAtArafah bool
}

// NewFastingUsecase requires both calendars.
func NewFastingUsecase(indonesiaCalendar, ummAlQuraCalendar domain.CalendarRepository) (*FastingUsecase, error) {
	if indonesiaCalendar == nil {
		return nil, errIndonesiaCalendarRequired
	}
	if ummAlQuraCalendar == nil {
		return nil, errUmmAlQuraCalendarRequired
	}

	return &FastingUsecase{
		indonesiaCalendar: indonesiaCalendar,
		ummAlQuraCalendar: ummAlQuraCalendar,
	}, nil
}

func (u *FastingUsecase) Classify(_ context.Context, request domain.ClassificationRequest) (domain.Classification, error) {
	input, err := parseRequest(request)
	if err != nil {
		return domain.Classification{}, err
	}

	result := domain.Classification{
		Date:            input.date.Format(time.DateOnly),
		Weekday:         weekdayName(request.Language, input.date.Weekday()),
		Hijri:           input.hijri,
		Verdict:         domain.VerdictOptional,
		Recommendations: make([]domain.Occurrence, 0),
		Restrictions:    make([]domain.Occurrence, 0),
		Notes:           make([]string, 0),
		Disclaimer:      disclaimer(request.Language),
	}

	applyRestrictions(&result, request.Language)
	if len(result.Restrictions) > 0 {
		return result, nil
	}

	isPilgrimOnArafah := input.isPilgrimAtArafah &&
		input.hijri.Month == 12 &&
		input.hijri.Day == 9
	if isPilgrimOnArafah {
		result.Notes = append(result.Notes, text(
			request.Language,
			"A pilgrim standing at Arafah is generally not recommended to fast that day; consult Hajj guidance.",
			"Jemaah haji yang sedang wukuf di Arafah pada umumnya tidak dianjurkan berpuasa pada hari tersebut; ikuti bimbingan ibadah haji.",
		))
		return result, nil
	}

	applyRecommendations(&result, input, request.Language)
	if len(result.Recommendations) > 0 {
		result.Verdict = domain.VerdictRecommended
	}

	return result, nil
}

func parseRequest(request domain.ClassificationRequest) (parsedRequest, error) {
	date, err := time.Parse(time.DateOnly, request.Date)
	if err != nil {
		return parsedRequest{}, invalidRequest(domain.CodeDateFormat, "date")
	}

	isOutOfRange := request.Hijri.Year < 1 ||
		request.Hijri.Month < 1 ||
		request.Hijri.Month > 12 ||
		request.Hijri.Day < 1 ||
		request.Hijri.Day > 30
	if isOutOfRange {
		return parsedRequest{}, invalidRequest(domain.CodeHijriRange, "hijri")
	}

	return parsedRequest{
		date:              date,
		hijri:             request.Hijri,
		isPilgrimAtArafah: request.IsPilgrimAtArafah,
	}, nil
}

func applyRestrictions(result *domain.Classification, language domain.Language) {
	hijri := result.Hijri
	switch {
	case hijri.Month == 9:
		result.Verdict = domain.VerdictObligatoryPeriod
		result.Restrictions = append(result.Restrictions, domain.Occurrence{
			Key:      "ramadan",
			Name:     "Ramadan",
			Evidence: "Qur'an 2:185",
		})
		result.Notes = append(result.Notes, text(
			language,
			"Ramadan fasting is obligatory for those accountable and able; it is not classified as a voluntary fast.",
			"Puasa Ramadan wajib bagi Muslim mukalaf yang mampu; puasa ini tidak diklasifikasikan sebagai puasa sunnah.",
		))
	case hijri.Month == 10 && hijri.Day == 1:
		result.Verdict = domain.VerdictProhibited
		result.Restrictions = append(result.Restrictions, domain.Occurrence{
			Key:      "eid_al_fitr",
			Name:     text(language, "Eid al-Fitr", "Idulfitri"),
			Evidence: "Sahih al-Bukhari 1990",
		})
	case hijri.Month == 12 && hijri.Day == 10:
		result.Verdict = domain.VerdictProhibited
		result.Restrictions = append(result.Restrictions, domain.Occurrence{
			Key:      "eid_al_adha",
			Name:     text(language, "Eid al-Adha", "Iduladha"),
			Evidence: "Sahih al-Bukhari 1990",
		})
	case hijri.Month == 12 && hijri.Day >= 11 && hijri.Day <= 13:
		result.Verdict = domain.VerdictProhibited
		result.Restrictions = append(result.Restrictions, domain.Occurrence{
			Key:      "days_of_tashriq",
			Name:     text(language, "Days of Tashriq", "Hari Tasyrik"),
			Evidence: "Sahih Muslim 1141a",
		})
		result.Notes = append(result.Notes, text(
			language,
			"A narrow Hajj exception exists for a pilgrim unable to find a sacrificial animal; seek qualified guidance.",
			"Terdapat pengecualian terbatas bagi jemaah haji yang tidak mendapatkan hewan kurban; mintalah bimbingan ahli yang kompeten.",
		))
	}
}

func applyRecommendations(
	result *domain.Classification,
	input parsedRequest,
	language domain.Language,
) {
	weekday := input.date.Weekday()
	if weekday == time.Monday || weekday == time.Thursday {
		name := text(language, "Monday fast", "Puasa Senin")
		if weekday == time.Thursday {
			name = text(language, "Thursday fast", "Puasa Kamis")
		}
		result.Recommendations = append(result.Recommendations, domain.Occurrence{
			Key:      "monday_thursday",
			Name:     name,
			Evidence: "Jami` at-Tirmidhi 747",
		})
	}

	if input.hijri.Day >= 13 && input.hijri.Day <= 15 {
		result.Recommendations = append(result.Recommendations, domain.Occurrence{
			Key:      "white_days",
			Name:     text(language, "Ayyam al-Bid (White Days)", "Ayyamul Bidh (Hari-hari Putih)"),
			Evidence: "Sunan an-Nasa'i 2420",
		})
	}

	if input.hijri.Month == 1 && input.hijri.Day >= 9 && input.hijri.Day <= 10 {
		name := text(language, "Companion day for Ashura", "Hari pendamping Asyura")
		evidence := "Sahih Muslim 1134b"
		if input.hijri.Day == 10 {
			name = text(language, "Ashura", "Asyura")
			evidence = "Sahih Muslim 1162b"
		}
		result.Recommendations = append(result.Recommendations, domain.Occurrence{
			Key:      "ashura",
			Name:     name,
			Evidence: evidence,
		})
	}

	if input.hijri.Month == 10 && input.hijri.Day >= 2 {
		result.Recommendations = append(result.Recommendations, domain.Occurrence{
			Key: "six_shawwal_candidate",
			Name: text(
				language,
				"Eligible day among six days of Shawwal",
				"Hari yang dapat dipilih untuk enam hari puasa Syawal",
			),
			Evidence: "Sahih Muslim 1164a",
		})
		result.Notes = append(result.Notes, text(
			language,
			"The API marks eligible Shawwal dates; the caller must track which six days are observed.",
			"API menandai tanggal Syawal yang dapat dipilih; pengguna perlu mencatat sendiri enam hari yang telah dijalankan.",
		))
	}

	if input.hijri.Month == 12 && input.hijri.Day >= 1 && input.hijri.Day <= 9 {
		result.Recommendations = append(result.Recommendations, domain.Occurrence{
			Key:      "first_nine_dhul_hijjah",
			Name:     text(language, "First nine days of Dhu al-Hijjah", "Sembilan hari pertama Zulhijah"),
			Evidence: "Sunan Abi Dawud 2437",
		})
	}

	if input.hijri.Month == 12 && input.hijri.Day == 9 {
		result.Recommendations = append(result.Recommendations, domain.Occurrence{
			Key:      "arafah",
			Name:     text(language, "Day of Arafah for a non-pilgrim", "Hari Arafah bagi yang tidak berhaji"),
			Evidence: "Sahih Muslim 1162b",
		})
	}
}

func (u *FastingUsecase) Rules(_ context.Context, language domain.Language) []domain.Rule {
	if language == domain.LanguageIndonesian {
		return rulesIndonesian()
	}

	return rulesEnglish()
}

func rulesEnglish() []domain.Rule {
	return []domain.Rule{
		{
			Key:         "monday_thursday",
			Name:        "Mondays and Thursdays",
			Kind:        "recommended",
			Calculation: "Gregorian weekday is Monday or Thursday.",
			Evidence: []string{
				"https://sunnah.com/tirmidhi:745",
				"https://sunnah.com/abudawud:2436",
				"https://sunnah.com/tirmidhi:747",
				"https://sunnah.com/muslim:1162b",
			},
			Caveat: "Both days rest on the Prophet's regular practice and on deeds being presented to Allah (Jami` at-Tirmidhi 745 and 747, Sunan Abi Dawud 2436). Sahih Muslim 1162b gives an additional reason for Monday.",
		},
		{
			Key:         "white_days",
			Name:        "Ayyam al-Bid",
			Kind:        "recommended",
			Calculation: "Hijri day 13, 14, or 15 each month, except where a higher-priority restriction applies.",
			Evidence: []string{
				"https://sunnah.com/nasai:2420",
				"https://sunnah.com/abudawud:2449",
			},
		},
		{
			Key:         "six_shawwal_candidate",
			Name:        "Six days of Shawwal",
			Kind:        "recommended",
			Calculation: "Any six days from 2 through the end of Shawwal; POST /days/{date} counts the days a caller sends in fasting_dates.",
			Evidence:    []string{"https://sunnah.com/muslim:1164a"},
			Caveat:      "Schools differ on details such as consecutiveness and combining intentions.",
		},
		{
			Key:         "ashura",
			Name:        "Ashura with a companion day",
			Kind:        "recommended",
			Calculation: "10 Muharram, preferably with 9 Muharram; 11 Muharram is commonly given as an alternative by scholars.",
			Evidence: []string{
				"https://sunnah.com/muslim:1134b",
				"https://sunnah.com/muslim:1162b",
			},
			Caveat: "The classifier emits only 9 and 10 Muharram because the cited primary text explicitly mentions adding the ninth.",
		},
		{
			Key:         "first_nine_dhul_hijjah",
			Name:        "First nine days of Dhu al-Hijjah",
			Kind:        "recommended",
			Calculation: "Hijri month 12, days 1-9; day 9 is Arafah.",
			Evidence:    []string{"https://sunnah.com/abudawud:2437"},
			Caveat:      "Sahih Muslim 1176 reports Aisha saying she never saw the Prophet fast these days; scholars reconcile the narrations differently, so ask a qualified scholar.",
		},
		{
			Key:         "arafah",
			Name:        "Day of Arafah",
			Kind:        "recommended",
			Calculation: "9 Dhu al-Hijjah for a non-pilgrim.",
			Evidence:    []string{"https://sunnah.com/muslim:1162b"},
			Caveat:      "This recommendation does not apply in the same way to a pilgrim standing at Arafah.",
		},
		{
			Key:         "shaban",
			Name:        "Frequent fasting in Sha'ban",
			Kind:        "recommended_pattern",
			Calculation: "A pattern of fasting much of Hijri month 8, not a fixed-date rule.",
			Evidence:    []string{"https://sunnah.com/bukhari:1969"},
			Caveat:      "Not emitted by date classification because it is a personal pattern and has additional jurisprudential details near Ramadan.",
		},
		{
			Key:         "three_monthly",
			Name:        "Three days each month",
			Kind:        "recommended_pattern",
			Calculation: "Any three days per Hijri month; days 13-15 are separately identified as Ayyam al-Bid.",
			Evidence: []string{
				"https://sunnah.com/bukhari:1981",
				"https://sunnah.com/nasai:2420",
			},
		},
		{
			Key:         "dawud",
			Name:        "Fast of Prophet Dawud",
			Kind:        "recommended_pattern",
			Calculation: "Alternate one fasting day and one non-fasting day, subject to prohibited days.",
			Evidence:    []string{"https://sunnah.com/bukhari:1976"},
			Caveat:      "Evaluated only by POST /days/{date} when the caller sets observes_dawud and sends recent fasting dates; the API never stores them.",
		},
		{
			Key:         "eid",
			Name:        "The two Eid days",
			Kind:        "prohibited",
			Calculation: "1 Shawwal and 10 Dhu al-Hijjah.",
			Evidence:    []string{"https://sunnah.com/bukhari:1990"},
		},
		{
			Key:         "tashriq",
			Name:        "Days of Tashriq",
			Kind:        "prohibited",
			Calculation: "11, 12, and 13 Dhu al-Hijjah.",
			Evidence:    []string{"https://sunnah.com/muslim:1141a"},
			Caveat:      "There is a narrow textual exception for certain pilgrims who cannot offer a sacrifice.",
		},
		{
			Key:         "friday_alone",
			Name:        "Friday alone",
			Kind:        "conditional_caution",
			Calculation: "Requires knowing adjacent fasting days and whether another established fast falls on Friday.",
			Evidence: []string{
				"https://sunnah.com/bukhari:1985",
				"https://sunnah.com/muslim:1144b",
			},
			Caveat: "Evaluated only by POST /days/{date} from the caller's fasting dates. A fast you customarily observe is excepted; ask a qualified scholar about individual circumstances.",
		},
	}
}
