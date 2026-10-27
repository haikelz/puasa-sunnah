# Changelog

All notable changes to the Puasa Sunnah API. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/). The API has no version segment in
its URLs, so breaking changes are listed here and in each release.

## [Unreleased]

### Added

- `GET /api/{language}/fasting/calendars` lists each calendar with its full coverage, publications, Sidang Isbat decisions, and caveat, so clients can warn before a date falls outside coverage.
- `calendar_source.isbat_decisions` lists Kemenag Sidang Isbat decisions for the months a response covers, with the PDF's planned start and the decided start. A decision replaces the planned month start. The 1447 H decisions for Ramadan, Syawal, and Zulhijah are recorded; all three matched the 2026 PDF.

## [1.1.0] - 2026-10-27

### Added

- An invariant test that walks every covered date of both calendars.
- CI on `dev` and pull requests, with golangci-lint, govulncheck, a gofmt check, and a Swagger drift check.
- Dependabot updates for Go modules, GitHub Actions, and Docker base images.

### Changed

- Logs are JSON, one object per line.
- `monday_thursday` recommendations name the weekday: "Monday fast" or "Thursday fast" in English, "Puasa Senin" or "Puasa Kamis" in Indonesian. The key is unchanged.
- The `monday_thursday` rule also cites Jami` at-Tirmidhi 745 and Sunan Abi Dawud 2436, and its caveat explains that both days rest on the Prophet's regular practice while Sahih Muslim 1162b gives an additional reason for Monday.

## [1.0.0] - 2026-10-25

First tagged release. Changes are relative to the untagged deployment before it.

### Breaking

- `/rules` keys `six_shawwal` and `dhul_hijjah` are now `six_shawwal_candidate` and `first_nine_dhul_hijjah`, matching the `types` filter.
- `calendar_source.id` for `calendar=id` is now `kemenag-ri`; the per-year ID moved to `calendar_source.publications[].id`.
- `POST /classify` accepts only JSON bodies; form bodies return `400`.
- Request bodies over 16 KiB return `413`, and requests over 5 seconds return `503`.

### Added

- `GET /api/{language}/fasting/days/{date}` classifies one date, restrictions included; `date` may be `today`.
- `POST /api/{language}/fasting/days/{date}` evaluates six Syawal progress, Dawud alternation, and the Friday-alone caution from history the caller sends, without storing it.
- `period=range` with `start` plus `end` or `days` for rolling feeds; ICS feeds gain a calendar name, a daily refresh interval, and optional `reminder=HH:MM` alarms.
- Kalender Hijriah Indonesia 2027; `calendar=id` covers 2025-12-28 through 2027-12-31.
- `Cache-Control` headers for CDN caching, an `X-Request-ID` header, and structured request logs.
- A weekly job that opens an issue when Elipski publishes the next Kemenag calendar.

### Changed

- Out-of-range `calendar=id` errors name `calendar=umm-al-qura` as an alternative.
- The first nine days of Dhu al-Hijjah carry a caveat citing Sahih Muslim 1176.
- Go 1.26, Fiber v2.52.15, and x/net v0.61.0.

### Fixed

- Requests rejected before routing, such as oversized bodies, no longer crash the server.
- `types` accepts spaces after commas.

[Unreleased]: https://github.com/haikelz/puasa-sunnah/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/haikelz/puasa-sunnah/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/haikelz/puasa-sunnah/releases/tag/v1.0.0
