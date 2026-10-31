# Puasa Sunnah API

A Go and Fiber API that classifies and schedules voluntary Islamic fasting days (_puasa sunnah_), in English and Indonesian.

> **Planning information, not a fatwa.** Hijri dates differ between countries, communities, and authorities. Use the date your local Islamic authority accepts, and ask a qualified scholar about personal rulings.

## Fasts it covers

| Fast | When | Evidence and notes |
| --- | --- | --- |
| Mondays and Thursdays | Every Monday and Thursday | The Prophet fasted both days regularly ([Jamiʿ at-Tirmidhi 745](https://sunnah.com/tirmidhi:745), [Sunan Abi Dawud 2436](https://sunnah.com/abudawud:2436)), and deeds are presented on them ([Jamiʿ at-Tirmidhi 747](https://sunnah.com/tirmidhi:747)). [Sahih Muslim 1162b](https://sunnah.com/muslim:1162b) gives an additional reason for Monday. |
| Ayyam al-Bid (White Days) | 13, 14, and 15 of each Hijri month | [Sahih al-Bukhari 1981](https://sunnah.com/bukhari:1981), [Sunan an-Nasa'i 2420](https://sunnah.com/nasai:2420). |
| Six days of Shawwal | Any six days from 2 Shawwal | [Sahih Muslim 1164a](https://sunnah.com/muslim:1164a). 1 Shawwal is Eid. Scholars differ on consecutiveness and combining intentions. |
| Ashura | 10 Muharram, with 9 Muharram | [Sahih Muslim 1134b](https://sunnah.com/muslim:1134b), [Sahih Muslim 1162b](https://sunnah.com/muslim:1162b). The API marks 9 and 10; scholars also mention the 11th. |
| First nine days of Dhu al-Hijjah | 1–9 Dhu al-Hijjah | [Sunan Abi Dawud 2437](https://sunnah.com/abudawud:2437). [Sahih Muslim 1176](https://sunnah.com/muslim:1176) reports Aisha never saw him fast these days; scholars reconcile the two differently. |
| Arafah | 9 Dhu al-Hijjah, for a non-pilgrim | [Sahih Muslim 1162b](https://sunnah.com/muslim:1162b). A pilgrim standing at Arafah is treated differently. |
| Much of Sha'ban | A personal pattern in month 8 | [Sahih al-Bukhari 1969](https://sunnah.com/bukhari:1969). Not marked on dates, because the days near Ramadan need care. |
| Fast of Dawud | Alternate a fast day and a rest day | [Sahih al-Bukhari 1976](https://sunnah.com/bukhari:1976). Needs your history, so only `POST /days` and `POST /patterns` evaluate it. |

### Restrictions come first

1. **Ramadan:** obligatory for those it applies to, so never labeled voluntary (Qur'an 2:185).
2. **Eid al-Fitr (1 Shawwal) and Eid al-Adha (10 Dhu al-Hijjah):** fasting is prohibited ([Sahih al-Bukhari 1990](https://sunnah.com/bukhari:1990)).
3. **Tashriq (11–13 Dhu al-Hijjah):** days of eating, drinking, and remembrance of Allah ([Sahih Muslim 1141a](https://sunnah.com/muslim:1141a)). A narrow exception exists for a pilgrim without a sacrificial animal.
4. **Friday alone:** discouraged unless you also fast Thursday or Saturday, or the Friday carries a fast you observe ([Sahih al-Bukhari 1985](https://sunnah.com/bukhari:1985), [Sahih Muslim 1144b](https://sunnah.com/muslim:1144b)). It needs your history, so only `POST /days` and `POST /patterns` report it.

Make-up fasts, vows, illness, travel, pregnancy, menstruation, intention, and differences between schools are outside a date classifier. Ask a qualified scholar.

## Calendars

A Hijri month starts on different days depending on the authority. In Indonesia, Kementerian Agama combines _hisab_ and _rukyat_ ([PMA 1/2026](https://kemenag.go.id/nasional/kemenag-terbitkan-pma-12026-padukan-hisab-dan-rukyat-dalam-penetapan-awal-bulan-hijriah-a94ay)). You choose the calendar with `calendar`:

- **`id` (default):** the Kalender Hijriah Indonesia from Ditjen Bimas Islam, [Tahun 2026](https://aceh.kemenag.go.id/storage/berita_informasi/01K7GE96M0ZVRJ62Z520RMYG1B.pdf) and [Tahun 2027](https://aceh.kemenag.go.id/storage/berita_informasi/01M2SQD5A9T6CWP76Y8A4PKB6J.pdf) (also on [Elipski](https://simbi.kemenag.go.id/eliterasi/)). Each PDF's month starts are typed by hand into `internal/domains/fasting/repository/kemenag/<year>.json` with the PDF's SHA-256. Coverage: 28 December 2025 to 31 December 2027.
  - Sidang Isbat decisions for Ramadan, Syawal, and Zulhijah go into the same file under `isbat_decisions`, with a link to the Kemenag announcement, and replace the PDF's planned start. The 1447 H decisions are recorded; all three matched the 2026 PDF.
  - The service refuses to start if the files disagree, leave a gap, or hold a month that isn't 29 or 30 days long.
- **`umm-al-qura`:** the Saudi civil planning calendar from [R.H. van Gent's data](https://webspace.science.uu.nl/~gent0113/islam/ummalqura.htm), 14 March 1937 to 16 November 2077. It is calculated, not an Indonesian religious determination.

A date outside the chosen calendar returns `422`; for `id`, the message suggests `calendar=umm-al-qura`. The API never switches calendars on its own. Every response carries `calendar_source`: publisher, coverage, status, a caveat, the PDFs behind the range, and the Isbat decisions for the months it covers. Every `id` response keeps the Isbat caveat, because some months may not have a decision yet.

Kemenag has no public calendar API. A weekly job (`.github/workflows/kemenag-calendar.yaml`) opens an issue when the next year's PDF appears on Elipski or when fewer than 60 days of coverage remain. A person adds the year; the job never edits data.

## API

Every business route starts with `/api/en` or `/api/id`. Keys, verdicts, and statuses stay the same in both languages; names, notes, caveats, and error messages are translated.

- Success: `{ "data": ... }`. Error: `{ "error": { "code": 400, "message": "..." } }`. Lists are `[]`, never `null`.
- Dates are strict `YYYY-MM-DD` in `Asia/Jakarta`. Weeks run Monday to Sunday.
- Each response has an `X-Request-ID` header matching the log line. Requests over 5 seconds return `503`, and bodies over 16 KiB return `413`.
- Caching: one day at the CDN (one hour in browsers) for an explicit date, five minutes for anything relative to today. Responses built from your history, errors, and `/health` are `no-store`. Each Vercel deploy clears the CDN.

| Endpoint | What it returns |
| --- | --- |
| `GET /health` | Service status |
| `GET /api/{lang}/fasting/rules` | Every rule with its calculation, evidence links, and caveats |
| `GET /api/{lang}/fasting/calendars` | Each calendar's coverage, PDFs, and Isbat decisions |
| `GET /api/{lang}/fasting/schedules` | Recommended days for a week, month, year, or range, as JSON or ICS |
| `GET /api/{lang}/fasting/days/{date}` | One date, restrictions included |
| `POST /api/{lang}/fasting/days/{date}` | One date plus personal patterns from your history |
| `POST /api/{lang}/fasting/patterns` | Personal patterns across a range |
| `POST /api/{lang}/fasting/classify` | One date, with a Hijri date you supply |

Full schemas: Swagger UI at `/swagger/index.html`, JSON at `/swagger/doc.json`.

### Schedules

```http
GET /api/id/fasting/schedules?period=month&date=2026-08-09
GET /api/en/fasting/schedules?period=year&date=2027-08-09&calendar=umm-al-qura&types=white_days
GET /api/id/fasting/schedules?period=range&days=90&format=ics&reminder=20:00
```

- `period`: `week`, `month`, or `year` around `date` (default today), or `range` with `start` (default today) plus `end` or `days`, up to 366 days.
- `types`: comma-separated keys to keep: `monday_thursday`, `white_days`, `ashura`, `six_shawwal_candidate`, `first_nine_dhul_hijjah`, `arafah`.
- `format=ics`: a calendar file with one all-day event per date. Omit `start` for a rolling feed that never runs out; subscribe over `webcal://` or "add calendar from URL". `reminder=HH:MM` adds an alarm on the evening before each fast.
- `is_pilgrim_at_arafah=true`: drops the Arafah fast for a pilgrim.

Schedules list only recommended days; Ramadan, Eid, and Tashriq never appear. Shawwal days are candidates; the schedule doesn't track your six.

### One date

```http
GET /api/id/fasting/days/today
GET /api/en/fasting/days/2027-03-10?calendar=umm-al-qura
```

Returns the verdict, recommendations, restrictions, and the Hijri date from the chosen calendar, so it answers "should I fast today?". `date` can be `today`.

To add personal patterns, POST your history to the same path:

```http
POST /api/en/fasting/days/2027-08-06
Content-Type: application/json

{ "fasting_dates": ["2027-08-04"], "observes_dawud": true }
```

`fasting_dates` lists days you fasted or plan to fast (at most 366). `observes_dawud` turns on the Dawud alternation. The response adds `patterns`:

| Key | Statuses | When |
| --- | --- | --- |
| `six_shawwal_candidate` | `in_progress`, `complete` | 2–30 Syawal. `progress` counts your Syawal days toward 6, and `eligible_days_left` counts the days left in the month. |
| `dawud` | `fast_day`, `rest_day` | You set `observes_dawud`. The alternation rests the day after a recorded fast. |
| `friday_alone` | `caution`, `paired`, `coincides` | A Friday. `paired`: you fast the Thursday or Saturday next to it. `coincides`: the Friday carries a fast you observe, such as Ayyamul Bidh or your Dawud day. |

Restricted days and a pilgrim's Arafah day carry no patterns. The API reads your history for that one request and never stores or logs it.

### Patterns across a range

```http
POST /api/id/fasting/patterns?start=2027-01-25&end=2027-02-12
Content-Type: application/json

{ "fasting_dates": ["2027-01-24"], "observes_dawud": true }
```

Takes the same body, plus `start` with `end` or `days` (up to 366), `calendar`, and `is_pilgrim_at_arafah`. `dates` lists only the days where a pattern applies.

- Six Syawal and Friday read only the dates you send. The plan never assumes a Syawal fast you didn't record.
- With `observes_dawud`, Dawud days are projected forward from your last recorded fast. The projection stops at the first Ramadan or prohibited day, where scholars differ on resuming, and `notes` names that date.

### Classify with your own Hijri date

```http
POST /api/en/fasting/classify
Content-Type: application/json

{ "date": "2026-06-25", "hijri": { "year": 1448, "month": 1, "day": 10 } }
```

```json
{
  "data": {
    "date": "2026-06-25",
    "weekday": "Thursday",
    "hijri": { "year": 1448, "month": 1, "day": 10 },
    "verdict": "recommended",
    "recommendations": [
      { "key": "monday_thursday", "name": "Thursday fast", "evidence": "Jami` at-Tirmidhi 747" },
      { "key": "ashura", "name": "Ashura", "evidence": "Sahih Muslim 1162b" }
    ],
    "restrictions": [],
    "notes": [],
    "disclaimer": "Planning information only..."
  }
}
```

`verdict` is `optional`, `recommended`, `obligatory_period`, or `prohibited`. It describes the calendar facts, not your personal ruling.

## Development

Needs Go 1.26 or later. [Air](https://github.com/air-verse/air) rebuilds on save:

```bash
go install github.com/air-verse/air@v1.65.3
air -c .air.toml        # or: go run ./cmd/api
curl http://localhost:8080/api/id/fasting/days/today
```

`.env` is not loaded automatically; export its variables yourself. Before committing:

```bash
go test ./...
go vet ./...
go build -o bin/api ./cmd/api
```

After changing routes or API models, regenerate Swagger and commit `internal/domains/swagger/generated/` (never edit it by hand):

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4
swag init -g main.go -d cmd/api,internal/app,internal/domains/fasting -o internal/domains/swagger/generated
```

The code is a modular monolith: `cmd/api` runs the process, `internal/app` wires it, and `internal/domains/fasting` holds the HTTP handlers, use cases, and the two calendar repositories. `AGENTS.md` lists the architecture and religious-accuracy rules.

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_NAME` | `puasa-sunnah-api` | Service name reported by `/health` |
| `PORT` | `8080` | HTTP port; Vercel sets it in production |

## Deployment

Production runs on [Vercel's Go runtime](https://vercel.com/docs/functions/runtimes/go). `vercel.json` builds `./cmd/api`, and Vercel supplies `PORT`. The service is stateless, so any future durable data needs an external store.

```bash
npx vercel login
npx vercel build        # check the production build locally
npx vercel              # preview
npx vercel --prod       # production
curl https://<your-deployment>/health
```

Set environment variables in the Vercel project, and never commit `.vercel/` or credentials.

Docker also works. `PORT` in `.env` picks the host port; inside the container the API listens on 8080. Pushes to `master` test the code and publish an image to GitHub Container Registry.

```bash
cp .env.example .env
docker network create shared-network   # once
docker compose up --build
```

## Changing a religious rule

1. Cite the primary evidence and the recognized scholarly reading.
2. Separate consensus, school-specific details, and implementation choices.
3. Add precedence and boundary tests in both languages.
4. Never present a projected Hijri date as an authoritative one.
5. Have qualified Islamic reviewers check material changes before release.

`GET /api/{lang}/fasting/rules` returns the evidence links. The English and Indonesian labels summarize the rules; they don't replace the Arabic texts or scholarly explanation.
