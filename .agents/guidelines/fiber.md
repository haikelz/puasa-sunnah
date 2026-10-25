# Fiber HTTP Service Coding-Agent Guideline

This document adds rules for building and changing HTTP services with Fiber.

Precedence follows `general.md` §2. Where `haikel-go.md` disagrees with this
guide, `haikel-go.md` wins, except for constructor rules: this guide governs
Fiber constructors.

Existing public routes, methods, status codes, headers, bodies, authentication
behavior, and operational endpoints are compatibility contracts unless the task
explicitly changes them.

## 1. Discover the Installed Fiber Version

Before editing, the agent MUST:

1. Read applicable instructions, `go.mod`, entry points, app construction, route
   registration, middleware, handlers, error mapping, configuration, tests, and
   generated API documentation.
2. Determine the Go and Fiber major versions from the module graph. Fiber v2 and
   v3 differ in import paths, context types, signatures, and middleware APIs.
3. Identify every runtime: HTTP server, worker, command, test app, serverless
   adapter, and prefork child.
4. Trace middleware order, groups, wildcard routes, mounts, static files, error
   handling, and trusted proxies before changing them.

Never upgrade Fiber, Go, middleware, or serializers as part of unrelated work.

## 2. Architecture and Constructors

```text
Fiber route and middleware → HTTP handler → usecase → domain contracts → repository or provider
```

- Fiber types stay in HTTP delivery, middleware, and wiring. Usecases and
  repositories never accept a Fiber context.
- Handlers parse transport input, call one usecase operation, and translate the
  result into the HTTP contract. Business policy belongs in usecases.
- One composition root constructs configuration, adapters, usecases, handlers,
  middleware, and routes. Never hide services or the app in mutable globals.
- Constructors validate required dependencies and return an error when one is
  missing, including typed nil values inside an interface. Fields stay
  unexported. Handlers expose a `Register(router fiber.Router)` method.

```go
var errUsecaseRequired = errors.New("order usecase is required")

type Handler struct {
	usecase order.Usecase
}

func NewHandler(usecase order.Usecase) (*Handler, error) {
	if isNil(usecase) {
		return nil, errUsecaseRequired
	}

	return &Handler{
		usecase: usecase,
	}, nil
}

func (h *Handler) Register(router fiber.Router) {
	router.Use(requireSupportedLanguage)
	router.Get("/orders", h.list)
	router.Get("/orders/:id", h.get)
	router.Post("/orders", h.create)
}
```

## 3. App Construction and Configuration

- Construct a fresh app per application or test instance from validated typed
  configuration. Never initialize the app as a package-level variable.
- Tests MUST create isolated app instances without depending on process
  environment or prior route registration.
- Do not enable `Prefork` by default. Enable it only after measuring a need and
  confirming compatibility with containers, state, telemetry, and shutdown.
- Set body, header, concurrency, and timeout limits deliberately.
- Configure trusted proxies to match the actual network before trusting
  forwarded addresses or schemes.

## 4. Routes and Groups

- Each module registers its routes through its handler's `Register` method.
- Register group middleware before the handlers it protects, and static routes
  before broad parameters or wildcards. Catch-all routes go last.
- Route methods, parameter names, and trailing-slash behavior are public
  contracts.
- Language, tenant, or version groups reject unsupported values before handlers
  run.
- Add a version prefix only for a real compatibility policy.

## 5. Middleware

A typical order: request ID, recovery, access logging and tracing, trusted proxy
handling, security headers and CORS, body limits, timeout where compatible,
authentication, then authorization and route-specific rate limits.

- Recovery wraps downstream code and keeps panics observable without exposing
  stack traces.
- Access logs never include credentials, authorization headers, cookies, or
  sensitive bodies, and use normalized route patterns.
- Declare CORS origins, methods, headers, and credentials explicitly.
- Add CSRF protection to cookie-authenticated browser mutations, not blindly to
  bearer-token APIs.
- Rate limits use a stable, proxy-aware key with bounded storage.
- Timeout middleware MUST NOT leave background work mutating state after the
  client receives a timeout.
- Authentication middleware MUST NOT block health endpoints the orchestrator
  needs.

## 6. Fiber Context and Request Lifetime

Fiber reuses contexts and request buffers. Treat every request-scoped value as
short-lived.

| Concern | v2 | v3 |
| --- | --- | --- |
| Context type | `*fiber.Ctx` | `fiber.Ctx` (interface) |
| `context.Context` for usecases | `c.UserContext()` | `c.Context()` |
| Values from params, query, headers, body | zero-copy; copy with `utils.CopyString` or set `Config.Immutable` before they escape | check the installed version's copy semantics |

- Never retain the Fiber context after the handler returns, and never capture it
  or request-backed byte slices in a goroutine.
- Pass the converted `context.Context` through usecases and I/O so cancellation
  and deadlines propagate.
- Fiber locals carry only transport-scoped data such as identity, request ID, or
  parsed route policy. Use constant keys.
- Do not assume `net/http` behavior where fasthttp differs.

## 7. Parsing and Validation

- Parse path, query, header, cookie, and body input explicitly into request DTOs,
  never persistence models.
- Malformed values MUST NOT silently become zero values or defaults.
- Struct tags are not proof that validation ran. Call the configured validator
  and map its result.
- Pagination validates positive bounds and computes offsets without overflow or
  slice panics.
- Give dates, timezones, sorting, filtering, and enums one authoritative parser.
- Uploads enforce count, size, media type, filename, and storage path before
  persistence.

## 8. Responses and Errors

- Follow the repository's envelope. Keep it consistent across equivalent routes.
- Status codes reflect the contract: malformed input is not `500`, missing
  resources are not implicitly successful, accepted asynchronous work is not
  completed work.
- Serialize empty collections as `[]` when the contract models a collection.
- Map domain sentinels with `errors.Is` or `errors.As` at one HTTP boundary. The
  central Fiber error handler is the final safe fallback for unknown errors.
- Never send internal errors, stack traces, SQL, provider payloads, or paths to
  clients.
- Write a response once and set headers and cookies before the body.

## 9. Authentication and Authorization

- Authentication middleware validates algorithm, signature, issuer, audience,
  expiry, and not-before as applicable.
- Check authorization separately for each protected operation and object scope.
- Never trust user IDs, roles, tenants, or permissions from the request body.
- Cookies define `Secure`, `HttpOnly`, `SameSite`, path, domain, and lifetime.
- A route documented as protected MUST have enforced middleware and tests.

## 10. Static Files, Redirects, and Proxies

- Fix static roots and protect them against traversal and dotfile access.
- Unknown static paths keep their `404`. Return the homepage with `200` only for
  a deliberate SPA.
- Precompressed files require correct `Accept-Encoding` negotiation,
  `Content-Encoding`, `Vary`, and MIME behavior.
- Allowlist redirect destinations derived from input.

## 11. API Documentation

- Every handler carries a `// <Method> godoc` block with `@Summary`, `@Tags`,
  `@Param`, `@Success`, `@Failure` for only the outcomes it can return, and an
  `@Router` that matches the registered path.
- Regenerate Swagger through the documented command; never hand-edit generated
  files.
- Describe actual envelopes instead of vague `object` placeholders.

## 12. Health, Logging, and Shared State

- Liveness answers whether the process can serve; readiness answers whether it
  should receive traffic. Keep both cheap and free of secrets.
- Probes request routes that exist, using tools present in the runtime image.
- Use the repository logger with structured output to stdout. Log errors once at
  the boundary that owns the response.
- Synchronize shared maps, caches, and counters. Prefork processes share no
  ordinary memory, so never assume in-memory locks, caches, or rate limits are
  process-wide.
- No fire-and-forget goroutines from handlers; move long work to a durable job.

## 13. Startup and Shutdown

- Validate configuration and listen addresses before startup, and return
  startup failure as a process failure.
- SIGINT and SIGTERM trigger a bounded, idempotent graceful shutdown shorter than
  the orchestrator grace period: mark readiness false, drain requests, stop
  workers, and close dependencies in reverse order.
- Reusable packages never call `os.Exit` or `log.Fatal`.
- Serverless adapters build immutable state once, not per request.

## 14. Testing

- Construct a fresh app per test and call it through Fiber's test API with
  bounded timeouts.
- Cover success contracts, malformed and boundary input, domain error mapping,
  empty collections, missing resources, authorization, middleware order,
  unknown routes, health endpoints, and panic recovery.
- Decode and compare typed response shapes instead of substring assertions.
- Test usecases without Fiber.

## 15. Verification and Deployment

Run the command list in `haikel-go.md` §8. Also confirm generated API
documentation is current and that health, readiness, and unknown routes return
expected statuses. Container rules live in `docker.md`.
