# Puasa Sunnah API

A production-grade Go/Fiber API for explaining, classifying, and scheduling voluntary fasting days in the Islamic calendar. The project uses a modular monolith with delivery, use-case, and repository boundaries while keeping Fiber isolated at the HTTP edge.

> **Religious scope:** this production service provides calendar-based planning information, not a fatwa. Hijri dates can differ between countries, communities, and recognized authorities. Always use the Hijri date accepted by the relevant local Islamic authority and ask qualified scholars about personal rulings.

## What is puasa sunnah?

`Sawm` means abstaining from the things that invalidate fasting, with intention, from true dawn (`fajr`) until sunset (`maghrib`). Ramadan fasting is obligatory for accountable Muslims who are able (Qur'an 2:183–185). _Puasa sunnah_ is voluntary fasting established or encouraged through the practice and teaching of Prophet Muhammad ﷺ. It is worship: the purpose is drawing nearer to Allah, not merely following a calendar or obtaining a health outcome.

The main recurring forms represented by this project are:

| Fast                              | Calendar rule                                                                                     | Core evidence / note                                                                                                                                                                                                                                             |
| --------------------------------- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Mondays and Thursdays             | Gregorian weekday Monday or Thursday                                                              | Deeds are presented on these days: [Jamiʿ at-Tirmidhi 747](https://sunnah.com/tirmidhi:747). Monday also has a distinct basis in [Sahih Muslim 1162b](https://sunnah.com/muslim:1162b).                                                                          |
| Three days monthly / Ayyam al-Bid | Any three days monthly; the White Days are Hijri 13, 14, 15                                       | [Sahih al-Bukhari 1981](https://sunnah.com/bukhari:1981), [Sunan an-Nasa'i 2420](https://sunnah.com/nasai:2420).                                                                                                                                                 |
| Six days of Shawwal               | Any six days from 2 Shawwal onward                                                                | 1 Shawwal is Eid and may not be fasted. The six need tracking across requests. See [Sahih Muslim 1164a](https://sunnah.com/muslim:1164a). Scholars differ on details such as consecutiveness and combining intentions.                                           |
| Ashura                            | 10 Muharram, preferably accompanied by 9 Muharram; scholars commonly mention 11 as an alternative | [Sahih Muslim 1134b](https://sunnah.com/muslim:1134b), [Sahih Muslim 1162b](https://sunnah.com/muslim:1162b). The classifier emits 9 and 10 because the cited primary text explicitly mentions adding the ninth; the 11th remains a documented scholarly caveat. |
| First nine of Dhu al-Hijjah       | Hijri month 12, days 1–9                                                                          | [Sunan Abi Dawud 2437](https://sunnah.com/abudawud:2437). [Sahih Muslim 1176](https://sunnah.com/muslim:1176) reports Aisha never saw him fast these days; scholars reconcile the narrations differently. |
| Arafah                            | 9 Dhu al-Hijjah for a non-pilgrim                                                                 | [Sahih Muslim 1162b](https://sunnah.com/muslim:1162b). A pilgrim standing at Arafah is treated differently.                                                                                                                                                      |
| Much of Sha'ban                   | A personal pattern during month 8, not a fixed date                                               | [Sahih al-Bukhari 1969](https://sunnah.com/bukhari:1969). Details near Ramadan require care, so the classifier does not emit every Sha'ban date.                                                                                                                 |
| Fast of Dawud                     | Alternate a fasting day and a non-fasting day                                                     | [Sahih al-Bukhari 1976](https://sunnah.com/bukhari:1976). It requires personal history and is not calculated by this stateless endpoint.                                                                                                                         |

### Days with higher-priority restrictions

The classifier applies restrictions before recommendations:

1. **Ramadan (month 9):** fasting is obligatory for those to whom it applies, so it is not labeled voluntary (Qur'an 2:185).
2. **Eid al-Fitr (1 Shawwal) and Eid al-Adha (10 Dhu al-Hijjah):** fasting is prohibited; see [Sahih al-Bukhari 1990](https://sunnah.com/bukhari:1990).
3. **Days of Tashriq (11–13 Dhu al-Hijjah):** days of eating, drinking, and remembrance of Allah; see [Sahih Muslim 1141a](https://sunnah.com/muslim:1141a). A narrow exception exists for a pilgrim unable to offer a sacrificial animal.
4. **Friday by itself:** this depends on adjacent days and whether an independently established fast falls on Friday; see [Sahih al-Bukhari 1985](https://sunnah.com/bukhari:1985). The stateless endpoint therefore documents, but does not calculate, this personal sequence rule.

Other questions—making up Ramadan, vows, illness, travel, pregnancy, menstruation/postnatal bleeding, ability, intention, breaking a voluntary fast, combining intentions, or disagreement between legal schools—are deliberately outside a date classifier. Refer them to a qualified scholar.

## How date calculation works

The Hijri calendar has 12 lunar months of 29 or 30 days. A tabular or astronomical Gregorian↔Hijri conversion is useful for planning, but an observed religious date can differ by a day because month starts are established under different accepted methods and authorities. In Indonesia, Kementerian Agama combines _hisab_ and _rukyat_ in its official process; see [PMA 1/2026](https://kemenag.go.id/nasional/kemenag-terbitkan-pma-12026-padukan-hisab-dan-rukyat-dalam-penetapan-awal-bulan-hijriah-a94ay).

For manual classification, this API uses a transparent two-calendar input:

1. The caller supplies an ISO Gregorian date (`YYYY-MM-DD`). Go derives its weekday, which determines Monday/Thursday.
2. The caller supplies the locally accepted Hijri year, month, and day. No hidden conversion is performed.
3. Prohibited/obligatory periods are checked first.
4. If no restriction applies, fixed-date and weekday recommendations are accumulated. One date can match several recommendations.
5. Stateful patterns (six completed Shawwal days, alternate-day fasting, three chosen monthly days) remain the caller's responsibility.

Schedule generation supports two explicitly selected calendar sources:

- `id` (default) uses the published Kalender Hijriah Indonesia from Ditjen Bimas Islam, Kementerian Agama RI: [Tahun 2026](https://aceh.kemenag.go.id/storage/berita_informasi/01K7GE96M0ZVRJ62Z520RMYG1B.pdf) and [Tahun 2027](https://aceh.kemenag.go.id/storage/berita_informasi/01M2SQD5A9T6CWP76Y8A4PKB6J.pdf) (also on [Elipski](https://simbi.kemenag.go.id/eliterasi/)). Their month boundaries are transcribed by hand into `internal/domains/fasting/repository/kemenag/<year>.json`, one file per PDF with its SHA-256; this is not an automatically extrapolated calculation. The service refuses to start if the files disagree on a boundary, leave a coverage gap, or contain a month that isn't 29 or 30 days long. Kemenag does not expose a documented public calendar API, so coverage is limited to the publications' visible dates, currently 28 December 2025 through 31 December 2027. Unsupported ranges return HTTP `422`.
- `umm-al-qura` uses the Saudi Umm al-Qura civil calendar conversion data compiled by [R.H. van Gent](https://webspace.science.uu.nl/~gent0113/islam/ummalqura.htm), covering 14 March 1937 through 16 November 2077. It is a calculated planning calendar, not an Indonesian religious determination, and can differ from dates accepted by local authorities.

Every schedule response identifies the selected calendar, source, publisher, coverage, status, and relevant authority caveat. The API never silently falls back from one calendar to the other. `calendar_source.publications` lists the PDFs behind the requested range; a week that crosses New Year cites both years. Add later `id` years only after Kemenag publishes the corresponding official calendar and the transcribed month boundaries have tests. The weekly `.github/workflows/kemenag-calendar.yaml` job watches Elipski for the next year's calendar and opens an issue when it appears, or when fewer than 60 days of coverage remain. It never edits the data itself.

## API

Every fasting endpoint requires a language segment. Use `/api/en` for English or `/api/id` for Indonesian. Stable machine-readable values such as keys, verdicts, periods, and status identifiers remain unchanged in both languages; names, weekdays, calculations, caveats, notes, disclaimers, and validation messages are localized.

### Health

```http
GET /health
```

### List supported rules

```http
GET /api/en/fasting/rules
GET /api/id/fasting/rules
```

Returns calculations, evidence links, and caveats, including rules that cannot be determined from one stateless request.

Rule keys for fixed-date recommendations match the recommendation keys that schedules emit and the `types` filter accepts.

### Get a weekly, monthly, or yearly schedule

```http
GET /api/en/fasting/schedules?period=week&date=2026-08-09
GET /api/en/fasting/schedules?period=month&date=2026-08-09
GET /api/en/fasting/schedules?period=year&date=2026-08-09
GET /api/id/fasting/schedules?period=year&date=2026-08-09&calendar=id
GET /api/id/fasting/schedules?period=year&date=2027-08-09&calendar=umm-al-qura
```

- `period` is required and must be `week`, `month`, or `year`.
- `date` is an optional anchor in `YYYY-MM-DD`; it defaults to today in `Asia/Jakarta`.
- `calendar` is optional and must be `id` or `umm-al-qura`; it defaults to `id`.
- `types` optionally filters by comma-separated recommendation keys: `monday_thursday`, `white_days`, `ashura`, `six_shawwal_candidate`, `first_nine_dhul_hijjah`, or `arafah`. Matching is OR; only matching recommendations are returned, with overlapping dates deduplicated. Omitted/empty means all; unsupported keys return `400`.
- Requests that exceed the 5-second deadline return `503`. Every response carries an `X-Request-ID` header that matches the server log entry.
- `format=json|ics` defaults to JSON. `ics` downloads an RFC 5545 calendar with one all-day event per matching date, localized titles, source URL, caveat, and disclaimer. Empty schedules export a valid calendar without events. Import into Google Calendar, Apple Calendar, or Outlook; this is a snapshot, not an automatic subscription or notification service.
- Weeks run Monday through Sunday. Month and year use Gregorian boundaries.
- Add `is_pilgrim_at_arafah=true` for the Arafah-specific pilgrim context.
- Responses contain only recommended opportunities; obligatory Ramadan and prohibited Eid/Tashriq dates are suppressed.
- Shawwal dates are labeled as **eligible candidates**. The API does not choose or track the six days for a person.

Filter White Days or download them as a calendar:

```http
GET /api/id/fasting/schedules?period=month&date=2026-08-09&types=white_days
GET /api/id/fasting/schedules?period=year&date=2027-08-09&calendar=umm-al-qura&types=white_days,monday_thursday&format=ics
```

Filters never override Ramadan/Eid/Tashriq restrictions or calendar coverage. ICS events use stable IDs per calendar, audience, and date, with exclusive next-day end dates. No alarms are added automatically.

Example for August 2026:

```bash
curl 'http://localhost:8080/api/en/fasting/schedules?period=month&date=2026-08-09'
```

The response includes 11 distinct dates: the month's Mondays/Thursdays and 13–15 Rabi al-Awwal. An overlap such as Thursday, 27 August carries both recommendations in one date record. The abbreviated response below omits `dates`.

```json
{
  "data": {
    "period": "month",
    "anchor_date": "2026-08-09",
    "start_date": "2026-08-01",
    "end_date": "2026-08-31",
    "timezone": "Asia/Jakarta",
    "calendar": "id",
    "calendar_source": {
      "id": "kemenag-ri",
      "status": "published_planning_calendar",
      "coverage_start": "2025-12-28",
      "coverage_end": "2027-12-31",
      "publications": [
        {
          "id": "kemenag-ri-2026",
          "name": "Kalender Hijriah Indonesia Tahun 2026",
          "coverage_start": "2025-12-28",
          "coverage_end": "2027-01-02"
        }
      ]
    },
    "total": 11
  }
}
```

### Classify a date

```http
POST /api/en/fasting/classify
Content-Type: application/json

{
  "date": "2026-06-25",
  "hijri": { "year": 1448, "month": 1, "day": 10 },
  "is_pilgrim_at_arafah": false
}
```

Example response (fields abbreviated):

```json
{
  "data": {
    "date": "2026-06-25",
    "weekday": "Thursday",
    "hijri": { "year": 1448, "month": 1, "day": 10 },
    "verdict": "recommended",
    "recommendations": [
      {
        "key": "monday_thursday",
        "name": "Monday or Thursday",
        "evidence": "Jami` at-Tirmidhi 747"
      },
      { "key": "ashura", "name": "Ashura", "evidence": "Sahih Muslim 1162b" }
    ],
    "restrictions": [],
    "notes": [],
    "disclaimer": "Planning information only..."
  }
}
```

`verdict` is one of `optional`, `recommended`, `obligatory_period`, or `prohibited`. It describes only the supplied calendar facts; it is not an individual legal ruling.

### Swagger / OpenAPI documentation

Run the API, then open:

- Interactive Swagger UI: `http://localhost:8080/swagger/index.html`
- Swagger JSON: `http://localhost:8080/swagger/doc.json`

The UI documents health, rules, classification, and week/month/year schedule endpoints, including request bodies, query enums, response schemas, errors, and both calendar-source caveats. “Try it out” is enabled.

After changing routes, annotations, or API models, regenerate and commit all generated artifacts:

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4
swag init -g main.go \
  -d cmd/api,internal/app,internal/domains/fasting \
  -o internal/domains/swagger/generated
```

Generated files are stored in `internal/domains/swagger/generated/` as Go, JSON, and YAML. Do not edit them manually.

## Architecture

```text
cmd/api (process lifecycle)
  └── internal/app (composition root and routes)
        └── fasting delivery/http (Fiber parsing and JSON)
              └── fasting.Usecase interface
                    └── fasting/usecase (rules and schedule orchestration)
                          └── CalendarRepository
                                ├── Kemenag Indonesia adapter (one JSON file per published year)
                                └── Umm al-Qura adapter
```

Architecture conventions:

- modules live under `internal/domains/<module>`;
- delivery, use-case, and repository responsibilities stay separate;
- constructors and explicit wiring are preferred over a DI framework;
- module contracts live at the module root;
- transport uses `context.Context` across the boundary—Fiber's `*fiber.Ctx` must not enter use cases;
- `internal/app` owns composition and `cmd/api` owns startup/shutdown;
- concrete persistence is added behind repository interfaces only when the domain needs it.

There is no database because the rules and bundled calendar data are deterministic. Calendar adapters form a repository boundary, allowing callers to explicitly choose an authority without changing fasting rules.

## Local development

Requirements: Go 1.24+. For automatic rebuilds during development, install [Air](https://github.com/air-verse/air):

```bash
go install github.com/air-verse/air@v1.65.3
```

```bash
cp .env.example .env       # optional; .env is not auto-loaded
export $(grep -v '^#' .env | xargs)  # if using the file
air -c .air.toml           # watches Go files, rebuilds, and restarts
```

Air uses `.air.toml`, writes its temporary binary under ignored `tmp/air/`, excludes test files and generated/dependency directories, sends an interrupt for graceful restart, and removes temporary files when it exits. Use `go run ./cmd/api` when live reload is not needed.

Then call:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/en/fasting/rules
curl http://localhost:8080/api/id/fasting/rules
curl 'http://localhost:8080/api/en/fasting/schedules?period=week&date=2026-08-09'
curl 'http://localhost:8080/api/id/fasting/schedules?period=month&date=2026-08-09'
curl 'http://localhost:8080/api/en/fasting/schedules?period=year&date=2026-08-09'
curl 'http://localhost:8080/api/id/fasting/schedules?period=year&date=2027-08-09&calendar=umm-al-qura'
curl -X POST http://localhost:8080/api/en/fasting/classify \
  -H 'Content-Type: application/json' \
  -d '{"date":"2026-06-25","hijri":{"year":1448,"month":1,"day":10}}'
```

Verification:

```bash
go test ./...
go vet ./...
go build -o bin/api ./cmd/api
```

## Docker

```bash
cp .env.example .env
docker network create shared-network   # once; compose expects this external network
docker compose up --build
```

`PORT` in `.env` picks the host port. Inside the container the API always listens on 8080, and the health check calls `/health` with busybox `wget`. Pushes to `master` run `.github/workflows/build.yaml`, which tests the code and then publishes the image to GitHub Container Registry.

## Vercel deployment

The API is deployed with [Vercel's Go runtime](https://vercel.com/docs/functions/runtimes/go). Vercel detects the root `go.mod` and `cmd/api/main.go`, builds the Fiber server, provides its runtime `PORT`, and serves the process as a Vercel Function. No Docker container or long-lived local filesystem is required.

```text
Internet → Vercel CDN → Go Function → Fiber on $PORT
```

The deployment contract lives in `vercel.json`: the framework is `go`, and the build command emits the `server` binary from `./cmd/api`. The service remains stateless because Vercel may replace instances between requests. Any future durable data must live in an external durable service.

Requirements: a Vercel account and Node.js for invoking the Vercel CLI. The Hobby plan can host personal, non-commercial projects within its included Function limits.

```bash
npx vercel login
npx vercel build       # local production-build validation
npx vercel             # preview deployment
npx vercel --prod      # production deployment
```

After deployment, use the URL printed by Vercel to verify the public service:

```bash
DEPLOYMENT_URL=https://your-project.vercel.app
curl "$DEPLOYMENT_URL/health"
curl "$DEPLOYMENT_URL/api/en/fasting/rules"
curl "$DEPLOYMENT_URL/api/id/fasting/schedules?period=year&calendar=id"
```

Swagger is available at `$DEPLOYMENT_URL/swagger/index.html`. Do not commit `.vercel/` or credentials. Configure environment variables in the Vercel project settings; `PORT` is supplied automatically.

## Configuration

| Variable   |            Default | Purpose                                                    |
| ---------- | -----------------: | ---------------------------------------------------------- |
| `APP_NAME` | `puasa-sunnah-api` | Fiber service name and health metadata                     |
| `PORT`     |             `8080` | HTTP port; Vercel supplies this automatically in production |

Configuration is loaded and validated once at startup. Domain packages must not read environment variables directly.

## Sources and maintenance rule

The API's evidence URLs are returned by `GET /api/{language}/fasting/rules`. Primary hadith links above are used for traceability. Schedule dates come from either Kemenag's official yearly PDFs or the explicitly requested Umm al-Qura dataset; both retain their source-specific planning caveats. English and Indonesian labels summarize rules and are not substitutes for the Arabic texts or scholarly explanation.

When changing a religious rule:

1. document the primary evidence and recognized scholarly interpretation;
2. distinguish consensus, school-specific details, and implementation policy;
3. add precedence and boundary tests;
4. never silently convert a projected Hijri date into an authoritative religious date;
5. have qualified Islamic reviewers validate material changes before production release.
