package fasting

type Language string

const (
	LanguageEnglish    Language = "en"
	LanguageIndonesian Language = "id"
)

func (language Language) IsSupported() bool {
	return language == LanguageEnglish || language == LanguageIndonesian
}

type HijriDate struct {
	Year  int `json:"year" example:"1448" binding:"required"`
	Month int `json:"month" example:"1" minimum:"1" maximum:"12" binding:"required"`
	Day   int `json:"day" example:"10" minimum:"1" maximum:"30" binding:"required"`
}

type ClassificationRequest struct {
	Date              string    `json:"date" example:"2026-06-25" format:"date" binding:"required"`
	Hijri             HijriDate `json:"hijri" binding:"required"`
	IsPilgrimAtArafah bool      `json:"is_pilgrim_at_arafah"`
	Language          Language  `json:"-" swaggerignore:"true"`
}

type Classification struct {
	Date            string       `json:"date"`
	Weekday         string       `json:"weekday"`
	Hijri           HijriDate    `json:"hijri"`
	Verdict         Verdict      `json:"verdict"`
	Recommendations []Occurrence `json:"recommendations"`
	Restrictions    []Occurrence `json:"restrictions"`
	Notes           []string     `json:"notes"`
	Disclaimer      string       `json:"disclaimer"`
}

type Verdict string

const (
	VerdictOptional         Verdict = "optional"
	VerdictRecommended      Verdict = "recommended"
	VerdictObligatoryPeriod Verdict = "obligatory_period"
	VerdictProhibited       Verdict = "prohibited"
)

type Occurrence struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Evidence string `json:"evidence"`
}

type Rule struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Calculation string   `json:"calculation"`
	Evidence    []string `json:"evidence"`
	Caveat      string   `json:"caveat,omitempty"`
}

type SchedulePeriod string

const (
	SchedulePeriodWeek  SchedulePeriod = "week"
	SchedulePeriodMonth SchedulePeriod = "month"
	SchedulePeriodYear  SchedulePeriod = "year"
)

type Calendar string

const (
	CalendarIndonesia Calendar = "id"
	CalendarUmmAlQura Calendar = "umm-al-qura"
)

type ScheduleRequest struct {
	Period            SchedulePeriod
	Types             []string
	Date              string
	IsPilgrimAtArafah bool
	Language          Language
	Calendar          Calendar
}

type Schedule struct {
	Period         SchedulePeriod `json:"period"`
	Calendar       Calendar       `json:"calendar"`
	AnchorDate     string         `json:"anchor_date"`
	StartDate      string         `json:"start_date"`
	EndDate        string         `json:"end_date"`
	WeekStartsOn   string         `json:"week_starts_on,omitempty"`
	Timezone       string         `json:"timezone"`
	Audience       string         `json:"audience"`
	CalendarSource CalendarSource `json:"calendar_source"`
	Total          int            `json:"total"`
	Dates          []ScheduleDate `json:"dates"`
	Disclaimer     string         `json:"disclaimer"`
}

type ScheduleDate struct {
	Date            string       `json:"date"`
	Weekday         string       `json:"weekday"`
	Hijri           HijriDate    `json:"hijri"`
	Recommendations []Occurrence `json:"recommendations"`
	Notes           []string     `json:"notes"`
}

type CalendarSource struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Publisher     string `json:"publisher"`
	URL           string `json:"url"`
	CoverageStart string `json:"coverage_start"`
	CoverageEnd   string `json:"coverage_end"`
	Status        string `json:"status"`
	Caveat        string `json:"caveat"`
}
