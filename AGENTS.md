# Engineering guide for Puasa Sunnah API

This file governs the whole repository. Preserve both technical correctness and religious humility. This is a production-grade API that provides calendar-based planning information; it does not issue fatwas or decide a user's personal obligation.

## Definition of done

For code changes, run the narrowest relevant tests, then before finishing run:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build -o bin/api ./cmd/api
```

For Vercel configuration changes also run `npx vercel build` when the Vercel CLI is available. Do not claim production deployment was tested unless an authenticated `npx vercel --prod` actually completed and the deployed health endpoint was checked.

Local development uses `air -c .air.toml`. Keep Air build output under ignored `tmp/air/`; do not point it at `bin/` or commit generated binaries. If the API entrypoint moves, update both `.air.toml` and `vercel.json`.

## Architecture constraints

- This is a modular monolith with module-owned delivery, use-case, and repository boundaries.
- `cmd/api` owns process lifecycle and graceful shutdown.
- `internal/app` is the composition root. Wire concrete dependencies explicitly there.
- Business modules live in `internal/domains/<module>`.
- A module root owns entities, DTOs, and narrow interfaces. Implementations belong in `usecase/`, `repository/`, and `delivery/http/` as needed.
- Fiber belongs only in delivery and app wiring. Never pass `*fiber.Ctx` to a use case or repository; pass `context.Context`.
- Use cases own rules and orchestration. Repositories own persistence/provider details. Do not add a repository for deterministic in-memory logic.
- Cross-module calls use narrow interfaces and constructor injection, never another module's concrete implementation or a service locator.
- Load environment configuration once through `internal/config`; domains must not call `os.Getenv`.
- Keep the API stateless. Vercel instances can be replaced between requests, and their bundled filesystem is read-only apart from temporary scratch space.
- Prefer the smallest implementation that preserves these boundaries. Do not create shared `utils`, `helpers`, or `models` catch-all packages.

When persistence becomes necessary, define the repository contract in the consuming module, keep SQL/database types out of the contract, use explicit versioned migrations, and make transaction boundaries match complete business state changes. Do not auto-migrate during API startup.

## HTTP conventions

- Prefix business routes with the required response language (`/api/en` or `/api/id`) and retain `/health` for platform probes. Do not add a version segment.
- Successful resources use `{ "data": ... }`; errors use `{ "error": { "code": <status>, "message": "..." } }`.
- Parse and validate transport input in delivery, but keep business/date validation in the use case.
- Return arrays as `[]`, not `null`.
- Derive weekdays from strict ISO `YYYY-MM-DD` dates.
- Keep Fiber-specific error mapping in delivery/app layers.
- Document every public endpoint with Swag annotations and concrete envelope schemas. After route or model changes, run `swag init -g main.go -d cmd/api,internal/app,internal/domains/fasting -o internal/domains/swagger/generated` and commit `internal/domains/swagger/generated/`.
- Swagger UI is served at `/swagger/index.html`; generated Swagger JSON is served at `/swagger/doc.json`. Never hand-edit generated Swagger files.

## Islamic-domain invariants

Religious accuracy has higher priority than convenience.

1. **Do not silently infer authoritative Hijri dates.** Computed calendars are estimates/planning tools. Month starts can differ by accepted authority and locality. Date classification requires an explicit Hijri date until a caller-selected, clearly disclosed calendar source exists.
2. **Restrictions precede recommendations.** Ramadan is an obligatory period, not voluntary. Eid al-Fitr (1 Shawwal), Eid al-Adha (10 Dhu al-Hijjah), and Tashriq (11–13 Dhu al-Hijjah) must suppress ordinary voluntary recommendations. Preserve documented exceptional cases as caveats rather than broad permissions.
3. **Separate fixed-date rules from personal patterns.** Monday/Thursday, Ayyam al-Bid, Ashura, Arafah, and fixed prohibited days can be classified from one request. Six completed Shawwal days, three selected monthly days, Dawud alternation, Friday paired with an adjacent day, and personal make-up obligations require history or context.
4. **Arafah requires pilgrim context.** Do not recommend the Arafah fast to a pilgrim standing at Arafah as though the same rule applied unconditionally.
5. **Do not turn disagreement into certainty.** Consecutive Shawwal days, combining intentions, Friday/Saturday details, local moon-sighting choices, and personal exemptions can involve jurisprudential differences. Label caveats and direct users to qualified scholars.
6. **Keep evidence traceable.** Material rules need a primary Qur'an/hadith reference and, where interpretation is not direct, recognized scholarly review. Do not use SEO calendar sites as the authority for production rules.
7. **Keep the disclaimer.** Outputs are planning information, not a fatwa. Do not promise reward, acceptance, or individual legal status.

Core rules and references are documented in `README.md` and returned by `GET /api/{language}/fasting/rules`. Any rule change requires boundary/precedence tests in both supported languages. Before production release, seek review from qualified Islamic scholars, especially for Indonesian users and Kementerian Agama calendar policy.

## Calendar data

- Schedule generation uses the official `Kalender Hijriah Indonesia Tahun 2026` PDF from Ditjen Bimas Islam. There is no documented public Kemenag calendar API; do not claim one exists.
- `internal/domains/fasting/repository/kemenag_calendar_repository.go` is intentionally year-bounded. Never extrapolate its month lengths or label a generic MABIMS calculation as official Kemenag data.
- Schedule callers may explicitly select `calendar=id|umm-al-qura`; omitted values default to `id`. Never silently fall back from one source to the other.
- Umm al-Qura is a Saudi civil/planning calendar. Preserve its data provenance, supported range, planning status, and warning that other accepted authorities and localities can differ.
- Every schedule response must expose source URL, publisher, coverage, planning status, and a source-appropriate caveat. Indonesian calendar responses must state that Sidang Isbat can supersede Ramadan, Shawwal, or Dhu al-Hijjah planning dates.
- Weeks are Monday–Sunday. Months and years use Gregorian boundaries in `Asia/Jakarta`.
- Add a new year only from a Kemenag publication. Transcribe every month boundary, add transition tests, verify the full Gregorian year, and update source metadata/coverage.
- Unsupported ranges must return an explicit error. Do not degrade to a different calendar authority.

## Vercel deployment constraints

- Production uses Vercel's Go Framework Preset, not a Docker container. Keep `go.mod` at the repository root and the executable entrypoint at `cmd/api/main.go` unless `vercel.json` is updated in the same change.
- The process must listen on Vercel's runtime-provided `PORT`. Never replace it with a fixed production port; the local default may remain 8080.
- Keep `vercel.json` configured with `"framework": "go"`. Its build must emit a runnable binary from `./cmd/api`.
- Do not commit `.vercel/` or credentials. Configure production environment variables through Vercel project settings or the Vercel CLI.
- Treat the runtime filesystem as ephemeral. Persist future durable data in an external service, not local files.
- Ask before running `npx vercel --prod` because it changes shared production infrastructure. `npx vercel build` is safe local validation.

<!-- AI-GUIDELINES:BEGIN -->
## AI Engineering Guidelines

Read `.agents/general.md` for every task. Read only the applicable companion guidelines below; repository-local contracts and instructions remain authoritative.

- `.agents/guidelines/go.md`
- `.agents/guidelines/docker.md`
<!-- AI-GUIDELINES:END -->
