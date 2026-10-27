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
	SchedulePeriodRange SchedulePeriod = "range"
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
	Start             string
	End               string
	Days              *int
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

const (
	CalendarStatusPublished  = "published_planning_calendar"
	CalendarStatusCalculated = "calculated_planning_calendar"
)

// CalendarCoverage is one selectable calendar with every publication and Isbat
// decision across its whole coverage.
type CalendarCoverage struct {
	Calendar       Calendar       `json:"calendar"`
	IsDefault      bool           `json:"is_default"`
	CalendarSource CalendarSource `json:"calendar_source"`
}

// CalendarSource describes a calendar series. CoverageStart and CoverageEnd span
// every supported date; Publications lists the documents behind the requested
// range, and IsbatDecisions the Sidang Isbat decisions for months it overlaps.
type CalendarSource struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Publisher      string                `json:"publisher"`
	URL            string                `json:"url"`
	CoverageStart  string                `json:"coverage_start"`
	CoverageEnd    string                `json:"coverage_end"`
	Status         string                `json:"status"`
	Caveat         string                `json:"caveat"`
	Publications   []CalendarPublication `json:"publications"`
	IsbatDecisions []IsbatDecision       `json:"isbat_decisions"`
}

// DtoDayRequest asks for one date. History is nil when the caller sent no
// personal context.
type DtoDayRequest struct {
	Date              string
	Calendar          Calendar
	IsPilgrimAtArafah bool
	Language          Language
	History           *DtoFastingHistory
}

// DtoFastingHistory is caller-supplied context for personal patterns. The API
// evaluates it for one request and never stores it.
type DtoFastingHistory struct {
	FastingDates  []string `json:"fasting_dates" example:"2027-08-05,2027-08-07"`
	ObservesDawud bool     `json:"observes_dawud"`
}

// DateToday asks for the current date in Asia/Jakarta wherever a date is accepted.
const DateToday = "today"

// Personal pattern keys reuse the /rules keys.
const (
	PatternSixShawwal  = "six_shawwal_candidate"
	PatternDawud       = "dawud"
	PatternFridayAlone = "friday_alone"
)

const (
	PatternStatusInProgress = "in_progress"
	PatternStatusComplete   = "complete"
	PatternStatusFastDay    = "fast_day"
	PatternStatusRestDay    = "rest_day"
	PatternStatusCaution    = "caution"
	PatternStatusPaired     = "paired"
	PatternStatusCoincides  = "coincides"
)

// Day is the classification of one date with its calendar source and the
// personal patterns the caller's history unlocks.
type Day struct {
	Classification
	Calendar       Calendar        `json:"calendar"`
	CalendarSource CalendarSource  `json:"calendar_source"`
	Patterns       []PatternStatus `json:"patterns"`
}

// PatternStatus reports a personal pattern that depends on the caller's history.
// It stays separate from the fixed-date recommendations.
type PatternStatus struct {
	Key      string           `json:"key"`
	Name     string           `json:"name"`
	Status   string           `json:"status"`
	Message  string           `json:"message"`
	Evidence string           `json:"evidence"`
	Caveat   string           `json:"caveat,omitempty"`
	Progress *PatternProgress `json:"progress,omitempty"`
}

// PatternProgress counts recorded days toward a pattern's target.
// EligibleDaysLeft counts the month's days from this date to its end, this
// date included.
type PatternProgress struct {
	Recorded         int `json:"recorded"`
	Target           int `json:"target"`
	EligibleDaysLeft int `json:"eligible_days_left"`
}

// DtoPatternRequest asks for personal patterns across a range: Start (default
// today) with End or Days, as in period=range. The API evaluates History for
// this request only and never stores it.
type DtoPatternRequest struct {
	Start             string
	End               string
	Days              *int
	Calendar          Calendar
	IsPilgrimAtArafah bool
	Language          Language
	History           DtoFastingHistory
}

// PatternPlan lists the dates in a range where a personal pattern applies. Six
// Shawwal and Friday read only the caller's dates; Dawud days are projected
// until Ramadan or a prohibited day interrupts them, and Notes says where.
type PatternPlan struct {
	Calendar       Calendar       `json:"calendar"`
	CalendarSource CalendarSource `json:"calendar_source"`
	StartDate      string         `json:"start_date"`
	EndDate        string         `json:"end_date"`
	Timezone       string         `json:"timezone"`
	Total          int            `json:"total"`
	Dates          []PatternDate  `json:"dates"`
	Notes          []string       `json:"notes"`
	Disclaimer     string         `json:"disclaimer"`
}

// PatternDate is one date with the personal patterns that apply to it.
type PatternDate struct {
	Date     string          `json:"date"`
	Weekday  string          `json:"weekday"`
	Hijri    HijriDate       `json:"hijri"`
	Patterns []PatternStatus `json:"patterns"`
}

// CalendarPublication is one document, such as a yearly Kemenag PDF, behind a
// calendar's dates.
type CalendarPublication struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	CoverageStart string `json:"coverage_start"`
	CoverageEnd   string `json:"coverage_end"`
}

// IsbatDecision is a Kemenag Sidang Isbat decision on when Ramadan, Shawwal, or
// Dhu al-Hijjah began. The calendar uses MonthStart in place of PlannedStart,
// the date the yearly PDF printed.
type IsbatDecision struct {
	HijriYear    int    `json:"hijri_year" example:"1447"`
	HijriMonth   int    `json:"hijri_month" example:"9"`
	PlannedStart string `json:"planned_start" example:"2026-02-19" format:"date"`
	MonthStart   string `json:"month_start" example:"2026-02-19" format:"date"`
	DecidedOn    string `json:"decided_on" example:"2026-02-17" format:"date"`
	URL          string `json:"url"`
}
