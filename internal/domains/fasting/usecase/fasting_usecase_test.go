package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	domain "puasa-sunnah/internal/domains/fasting"
	fastingRepository "puasa-sunnah/internal/domains/fasting/repository"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		request             domain.ClassificationRequest
		wantVerdict         domain.Verdict
		wantRecommendations []string
		wantRestrictions    []string
	}{
		{
			name: "ashura on thursday combines recommendations",
			request: domain.ClassificationRequest{
				Date: "2026-06-25", Hijri: domain.HijriDate{Year: 1448, Month: 1, Day: 10},
			},
			wantVerdict:         domain.VerdictRecommended,
			wantRecommendations: []string{"monday_thursday", "ashura"},
		},
		{
			name: "eid restriction takes precedence",
			request: domain.ClassificationRequest{
				Date: "2026-03-21", Hijri: domain.HijriDate{Year: 1447, Month: 10, Day: 1},
			},
			wantVerdict:      domain.VerdictProhibited,
			wantRestrictions: []string{"eid_al_fitr"},
		},
		{
			name: "ramadan is not voluntary",
			request: domain.ClassificationRequest{
				Date: "2026-02-23", Hijri: domain.HijriDate{Year: 1447, Month: 9, Day: 6},
			},
			wantVerdict:      domain.VerdictObligatoryPeriod,
			wantRestrictions: []string{"ramadan"},
		},
		{
			name: "pilgrim does not receive overlapping Arafah recommendation",
			request: domain.ClassificationRequest{
				Date: "2026-06-01", Hijri: domain.HijriDate{Year: 1447, Month: 12, Day: 9}, IsPilgrimAtArafah: true,
			},
			wantVerdict: domain.VerdictOptional,
		},
		{
			name: "tashriq suppresses a white day recommendation",
			request: domain.ClassificationRequest{
				Date: "2026-05-30", Hijri: domain.HijriDate{Year: 1447, Month: 12, Day: 13},
			},
			wantVerdict:      domain.VerdictProhibited,
			wantRestrictions: []string{"days_of_tashriq"},
		},
	}

	service := newCalendarUsecase(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := service.Classify(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}
			if got.Verdict != tt.wantVerdict {
				t.Errorf("Verdict = %q, want %q", got.Verdict, tt.wantVerdict)
			}
			assertKeys(t, got.Recommendations, tt.wantRecommendations)
			assertKeys(t, got.Restrictions, tt.wantRestrictions)
		})
	}
}

func TestClassifyRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := newCalendarUsecase(t).Classify(context.Background(), domain.ClassificationRequest{
		Date: "25-06-2026", Hijri: domain.HijriDate{Year: 1448, Month: 1, Day: 10},
	})
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("Classify() error = %v, want ErrInvalidRequest", err)
	}
}

func TestNewFastingUsecaseRequiresCalendars(t *testing.T) {
	t.Parallel()

	indonesiaCalendar, err := fastingRepository.NewKemenagCalendarRepository()
	if err != nil {
		t.Fatal(err)
	}

	ummAlQuraCalendar := fastingRepository.NewUmmAlQuraCalendarRepository()
	tests := []struct {
		name      string
		indonesia domain.CalendarRepository
		ummAlQura domain.CalendarRepository
		want      error
	}{
		{
			name:      "nil Indonesia calendar",
			ummAlQura: ummAlQuraCalendar,
			want:      errIndonesiaCalendarRequired,
		},
		{
			name:      "nil Umm al-Qura calendar",
			indonesia: indonesiaCalendar,
			want:      errUmmAlQuraCalendarRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, err := NewFastingUsecase(tt.indonesia, tt.ummAlQura)

			if service != nil {
				t.Errorf("NewFastingUsecase() service = %#v, want nil", service)
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("NewFastingUsecase() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func assertKeys(t *testing.T, occurrences []domain.Occurrence, want []string) {
	t.Helper()
	if len(occurrences) != len(want) {
		t.Fatalf("occurrence count = %d, want %d: %#v", len(occurrences), len(want), occurrences)
	}
	for i := range want {
		if occurrences[i].Key != want[i] {
			t.Errorf("occurrence[%d].Key = %q, want %q", i, occurrences[i].Key, want[i])
		}
	}
}

func TestRulesExposeFilterKeysInBothLanguages(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)
	filterKeys := []string{
		"monday_thursday",
		"white_days",
		"ashura",
		"six_shawwal_candidate",
		"first_nine_dhul_hijjah",
		"arafah",
	}

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		rules := make(map[string]domain.Rule)
		for _, rule := range service.Rules(context.Background(), language) {
			rules[rule.Key] = rule
		}

		for _, key := range filterKeys {
			if _, ok := rules[key]; !ok {
				t.Errorf("%s rules missing schedule filter key %q", language, key)
			}
		}
		if rules["first_nine_dhul_hijjah"].Caveat == "" {
			t.Errorf("%s first_nine_dhul_hijjah rule has no caveat", language)
		}
	}
}

func TestRulesDescribePersonalPatterns(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		rules := make(map[string]domain.Rule)
		for _, rule := range service.Rules(context.Background(), language) {
			rules[rule.Key] = rule
		}

		friday := rules[domain.PatternFridayAlone]
		if !strings.Contains(strings.Join(friday.Evidence, " "), "muslim:1144b") {
			t.Errorf("%s friday_alone evidence = %v, want Sahih Muslim 1144b", language, friday.Evidence)
		}
		descriptions := map[string]string{
			domain.PatternSixShawwal:  rules[domain.PatternSixShawwal].Calculation,
			domain.PatternDawud:       rules[domain.PatternDawud].Caveat,
			domain.PatternFridayAlone: friday.Caveat,
		}
		for key, description := range descriptions {
			if !strings.Contains(description, "POST /days/{date}") {
				t.Errorf("%s %s does not point to POST /days/{date}: %q", language, key, description)
			}
		}
	}
}

func TestMondayThursdayNamesTheWeekday(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)
	tests := []struct {
		date           string
		wantEnglish    string
		wantIndonesian string
	}{
		{
			date:           "2026-08-03",
			wantEnglish:    "Monday fast",
			wantIndonesian: "Puasa Senin",
		},
		{
			date:           "2026-08-06",
			wantEnglish:    "Thursday fast",
			wantIndonesian: "Puasa Kamis",
		},
	}

	for _, tt := range tests {
		wantNames := map[domain.Language]string{
			domain.LanguageEnglish:    tt.wantEnglish,
			domain.LanguageIndonesian: tt.wantIndonesian,
		}
		for language, want := range wantNames {
			got, err := service.Classify(context.Background(), domain.ClassificationRequest{
				Date: tt.date,
				// 5 Safar carries no other recommendation, so only the weekday applies.
				Hijri: domain.HijriDate{
					Year:  1448,
					Month: 2,
					Day:   5,
				},
				Language: language,
			})
			if err != nil {
				t.Fatalf("%s/%s Classify() error = %v", tt.date, language, err)
			}

			isNamed := len(got.Recommendations) == 1 &&
				got.Recommendations[0].Key == "monday_thursday" &&
				got.Recommendations[0].Name == want
			if !isNamed {
				t.Errorf("%s/%s recommendations = %+v, want monday_thursday named %q", tt.date, language, got.Recommendations, want)
			}
		}
	}
}

func TestMondayThursdayRuleExplainsEvidence(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)
	wantEvidence := []string{
		"https://sunnah.com/tirmidhi:745",
		"https://sunnah.com/abudawud:2436",
		"https://sunnah.com/tirmidhi:747",
		"https://sunnah.com/muslim:1162b",
	}

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		var rule domain.Rule
		for _, candidate := range service.Rules(context.Background(), language) {
			if candidate.Key == "monday_thursday" {
				rule = candidate
			}
		}

		if strings.Join(rule.Evidence, " ") != strings.Join(wantEvidence, " ") {
			t.Errorf("%s monday_thursday evidence = %v, want %v", language, rule.Evidence, wantEvidence)
		}
		// The caveat must say why Monday cites an extra hadith, so the list does not
		// read as Thursday having weaker support.
		if !strings.Contains(rule.Caveat, "745") || !strings.Contains(rule.Caveat, "1162b") {
			t.Errorf("%s monday_thursday caveat = %q, want it to explain the shared and Monday evidence", language, rule.Caveat)
		}
	}
}
