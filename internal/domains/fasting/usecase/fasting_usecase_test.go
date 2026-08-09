package usecase

import (
	"context"
	"errors"
	"testing"

	"puasa-sunnah/internal/domains/fasting"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		request             fasting.ClassificationRequest
		wantVerdict         fasting.Verdict
		wantRecommendations []string
		wantRestrictions    []string
	}{
		{
			name: "ashura on thursday combines recommendations",
			request: fasting.ClassificationRequest{
				Date: "2026-06-25", Hijri: fasting.HijriDate{Year: 1448, Month: 1, Day: 10},
			},
			wantVerdict:         fasting.VerdictRecommended,
			wantRecommendations: []string{"monday_thursday", "ashura"},
		},
		{
			name: "eid restriction takes precedence",
			request: fasting.ClassificationRequest{
				Date: "2026-03-21", Hijri: fasting.HijriDate{Year: 1447, Month: 10, Day: 1},
			},
			wantVerdict:      fasting.VerdictProhibited,
			wantRestrictions: []string{"eid_al_fitr"},
		},
		{
			name: "ramadan is not voluntary",
			request: fasting.ClassificationRequest{
				Date: "2026-02-23", Hijri: fasting.HijriDate{Year: 1447, Month: 9, Day: 6},
			},
			wantVerdict:      fasting.VerdictObligatoryPeriod,
			wantRestrictions: []string{"ramadan"},
		},
		{
			name: "pilgrim does not receive overlapping Arafah recommendation",
			request: fasting.ClassificationRequest{
				Date: "2026-06-01", Hijri: fasting.HijriDate{Year: 1447, Month: 12, Day: 9}, IsPilgrimAtArafah: true,
			},
			wantVerdict: fasting.VerdictOptional,
		},
		{
			name: "tashriq suppresses a white day recommendation",
			request: fasting.ClassificationRequest{
				Date: "2026-05-30", Hijri: fasting.HijriDate{Year: 1447, Month: 12, Day: 13},
			},
			wantVerdict:      fasting.VerdictProhibited,
			wantRestrictions: []string{"days_of_tashriq"},
		},
	}

	service := NewFastingUsecase(nil)
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

	_, err := NewFastingUsecase(nil).Classify(context.Background(), fasting.ClassificationRequest{
		Date: "25-06-2026", Hijri: fasting.HijriDate{Year: 1448, Month: 1, Day: 10},
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Classify() error = %v, want ErrInvalidRequest", err)
	}
}

func assertKeys(t *testing.T, occurrences []fasting.Occurrence, want []string) {
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
