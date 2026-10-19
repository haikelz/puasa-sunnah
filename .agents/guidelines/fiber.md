# Fiber HTTP Service Coding-Agent Guideline

This document is the Fiber-specific companion to the Go coding-agent guideline.
The Go guide remains authoritative for formatting, package design, context,
errors, concurrency, resource ownership, tests, and general verification. This
guide adds rules for building and changing HTTP services with Fiber. Apply
repository-local contracts and instructions first.

The terms **MUST**, **SHOULD**, and **AVOID** are normative. Existing public
routes, methods, status codes, headers, bodies, authentication behavior, and
operational endpoints are compatibility contracts unless the task explicitly
changes them.

## 1. Discover the Installed Fiber Version

Before editing, the agent **MUST**:

1. Read applicable instructions, `go.mod`, `go.sum`, entry points, app
   construction, route registration, middleware, handlers, error mapping,
   configuration, tests, generated API documentation, container files, and
   deployment manifests.
2. Determine the installed Go and Fiber major versions from the module graph.
   Legacy unversioned Fiber, v2, and v3 have different import paths, signatures,
   configuration, and middleware APIs.
3. Identify every runtime: HTTP server, worker, command, test app, serverless
   adapter, prefork child, and development reloader.
4. Trace middleware order, route groups, wildcard routes, mounts, static files,
   custom error handling, trusted proxies, and operational endpoints before
   changing them.
5. Identify the package that owns business policy, persistence, authentication,
   authorization, and response contracts. Fiber is a transport, not the domain
   architecture.
6. Check version-matched official documentation before using behavior that may
   differ by major version. Do not add compatibility casts or copy examples from
   another major version.

The agent **MUST NOT** silently upgrade Fiber, Go, middleware, serializers, or an
HTTP adapter as part of unrelated work.

## 2. Architecture and Dependency Direction

Use a clear dependency direction:

```text
Fiber route and middleware
          │
          ▼
HTTP handler / delivery adapter
          │
          ▼
Application use case
          │
          ▼
Domain contracts
          │
          ▼
Repository / provider adapter
```

- Fiber types **MUST** remain in HTTP delivery, middleware, and application
  wiring unless an explicitly transport-specific package owns them.
- Use cases, domain services, and repositories **MUST NOT** accept Fiber request
  context types.
- Handlers SHOULD parse transport input, invoke one application operation, and
  translate its result into the HTTP contract.
- Business validation and policy belong in use cases or domain types, not route
  callbacks.
- Repository interfaces SHOULD be narrow, domain-oriented, and defined by the
  consuming package.
- Dependencies **MUST** be passed through constructors. Do not hide services,
  repositories, configuration, or the app in mutable globals.
- One composition root SHOULD construct configuration, adapters, use cases,
  handlers, middleware, and routes.
- Packages named `utils`, `common`, or `helpers` **MUST NOT** become dumping
  grounds for unrelated transport, environment, data, and process behavior.

## 3. App Construction and Configuration

Construct a fresh app for each application or test instance:

```go
type Dependencies struct {
	Logger *slog.Logger
	Clock  Clock
}

func NewApp(cfg Config, deps Dependencies) (*fiber.App, error) {
	if deps.Logger == nil {
		return nil, errors.New("logger is required")
	}

	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: newErrorHandler(deps.Logger),
	})

	registerMiddleware(app, cfg, deps)
	registerRoutes(app, cfg, deps)
	return app, nil
}
```

- `fiber.Config` **MUST** be created from validated typed configuration.
- The app **MUST NOT** be initialized as a package-level variable.
- Repeated constructor calls **MUST NOT** reuse or mutate one global app.
- Tests **MUST** be able to create isolated app instances without depending on
  process-global environment or prior route registration.
- `Prefork` **MUST NOT** be enabled by default. Enable it only after measuring a
  production need and confirming compatibility with containers, process
  supervision, state, sockets, telemetry, and shutdown.
- Body, header, parameter, concurrency, and timeout limits **MUST** be deliberate
  and appropriate to exposed endpoints.
- Immutable settings SHOULD be read once at startup. Runtime code SHOULD receive
  typed values rather than repeatedly reading environment variables.
- Custom JSON encoders and decoders require compatibility tests for numbers,
  HTML escaping, invalid input, and response contracts.
- Trusted proxy configuration **MUST** match the actual network topology before
  forwarded client addresses or schemes are trusted.

## 4. Route Ownership, Groups, and Ordering

- Route registration SHOULD be centralized by module or transport package.
- A module SHOULD expose one registration function that receives its handler and
  middleware dependencies.
- Related routes SHOULD use groups for shared path prefixes and policy.
- Group middleware **MUST** be registered before the handlers it protects.
- Static routes **MUST** be registered before broad parameters or wildcards when
  ordering can affect matching.
- Catch-all routes SHOULD be last and tested against every neighboring route
  class.
- Route names, methods, parameter names, and trailing-slash behavior are public
  contracts.
- Removed or intentionally unsupported routes SHOULD have negative tests when
  accidental reintroduction is plausible.
- Version prefixes SHOULD represent a real compatibility policy. Do not add
  `/v1` merely as decoration.
- Language, tenant, or API-version groups **MUST** reject unsupported values
  before business handlers run.
- Mounting sub-apps **MUST** preserve middleware, error, path, and lifecycle
  semantics intentionally.

Prefer explicit handler methods over large anonymous callbacks:

```go
func (h *Handler) Register(router fiber.Router) {
	group := router.Group("/schedules", h.requireLanguage)
	group.Get("/", h.list)
	group.Get("/:date", h.getByDate)
}
```

## 5. Middleware as Executable Policy

Middleware order changes behavior and **MUST** be reviewed as policy. A typical
order is:

1. Request or correlation ID.
2. Panic recovery.
3. Access logging and tracing.
4. Trusted proxy and scheme handling.
5. Security headers and deliberate CORS.
6. Body and request-size limits.
7. Request timeout where compatible with the operation.
8. Compression where measurement justifies it.
9. Authentication.
10. Authorization, tenant, and route-specific rate limits.
11. Handler.

Exact ordering depends on the contract, but the following rules always apply:

- Recovery **MUST** wrap downstream code and keep panics observable without
  exposing stack traces to clients.
- Request IDs SHOULD be available to access logs, traces, errors, and responses.
- Access logs **MUST NOT** include credentials, authorization headers, cookies,
  sensitive bodies, or uncontrolled high-cardinality fields.
- CORS **MUST** declare allowed origins, methods, headers, credentials, and cache
  duration according to browser clients. Permissive defaults require an explicit
  public-API decision.
- CSRF protection applies to cookie-authenticated browser mutations. It SHOULD
  NOT be added blindly to stateless bearer-token or read-only APIs.
- Compression SHOULD avoid already compressed media and tiny responses.
- Rate limits **MUST** have a stable client key, bounded storage, proxy-aware
  identity, and distributed semantics compatible with deployment scale.
- Timeout middleware **MUST NOT** create background work that continues mutating
  state after the client receives a timeout.
- Middleware requiring authentication **MUST NOT** accidentally protect health
  endpoints needed by the orchestrator or expose operational endpoints intended
  to be private.
- Middleware behavior and ordering SHOULD have focused integration tests.

## 6. Fiber Context and Request Lifetime

Fiber optimizes request processing by reusing context and buffer-backed values.
Treat all request-scoped values as short-lived. In this section, "Fiber request
context" means `*fiber.Ctx` in v2 or the corresponding context type in the
installed major version.

- A handler **MUST NOT** retain the Fiber request context after it returns.
- Values derived from params, query strings, headers, cookies, body, or locals
  **MUST NOT** escape the request lifetime unless copied according to the
  installed Fiber version's semantics.
- Goroutines **MUST NOT** capture the Fiber request context or request-backed
  byte slices.
- Convert to `context.Context` at the transport boundary using the API supported
  by the installed Fiber major version.
- Deadlines, cancellation, and request-scoped metadata SHOULD propagate through
  use cases and I/O.
- Long loops and provider calls **MUST** observe cancellation at useful
  boundaries.
- Fiber locals SHOULD carry only transport-scoped data such as authenticated
  identity, request ID, or parsed route policy. They are not a dependency
  container.
- Keys for locals **MUST** avoid collisions through constants or private typed
  conventions appropriate to the version.
- Do not assume standard-library `net/http` behavior where Fiber or fasthttp has
  different semantics.

## 7. Parsing and Validation

Every request boundary is untrusted.

- Path, query, header, cookie, and body input **MUST** be parsed explicitly.
- Malformed values **MUST NOT** silently become zero values or defaults.
- Missing, malformed, out-of-range, and unsupported values SHOULD produce stable
  client errors with field-level context where safe.
- Syntactic transport parsing belongs in the handler; business invariants belong
  in a use case or domain type.
- Struct tags alone are not proof that validation ran. The handler **MUST** call
  the configured validator and map its result.
- Body parsing **MUST** use explicit request DTOs, not persistence models.
- Unknown-field policy SHOULD be deliberate for compatibility and typo
  detection.
- Pagination **MUST** validate positive bounds, calculate offsets safely, clamp
  or reject values deliberately, and avoid integer overflow or slice panics.
- Date, timezone, locale, sorting, filtering, and enum rules **MUST** have one
  authoritative parser.
- Uploads **MUST** enforce count, size, media type, filename, storage path, and
  scanning policy before persistence.
- Validation errors SHOULD preserve a stable machine-readable code and a useful
  human message without exposing internal implementation details.

## 8. Response and Error Contracts

Define one deliberate response policy:

```json
{
  "data": {}
}
```

```json
{
  "error": {
    "code": "invalid_date",
    "message": "date must use YYYY-MM-DD"
  }
}
```

- Success and error envelopes **MUST** remain consistent across equivalent
  routes.
- Status codes **MUST** reflect the contract: malformed input is not 500, missing
  resources are not implicitly successful, and accepted asynchronous work is
  not complete work.
- Empty collections SHOULD serialize as `[]` when the API contract models a
  collection. Initialize slices deliberately.
- Missing-resource behavior **MUST** explicitly choose 404, nullable success, or
  another documented representation.
- Internal errors **MUST NOT** be sent directly to clients.
- Stack traces, SQL, provider payloads, filesystem paths, and secret values
  **MUST NOT** appear in public errors.
- Domain errors SHOULD use stable sentinels or typed errors, wrap causes with
  `%w`, and be mapped at one HTTP boundary using `errors.Is` or `errors.As`.
- The central Fiber error handler SHOULD provide the final safe fallback for
  unknown errors and framework errors.
- A response **MUST** be written once. After sending or returning an error, code
  MUST stop executing the response path.
- Headers and cookies **MUST** be set before the body is committed.
- Content type, charset, cache policy, and serialization behavior are part of
  the response contract.

## 9. Handlers and Use Cases

A handler SHOULD follow a predictable sequence:

1. Read authenticated and route context.
2. Parse transport input.
3. Perform syntactic validation.
4. Construct a typed command or query.
5. Call one use-case operation with `context.Context`.
6. Map the result or domain error.
7. Return the response.

Rules:

- Handlers **MUST NOT** issue ad hoc database queries when a repository or use
  case owns the operation.
- Handlers **MUST NOT** contain deterministic business calculations simply
  because the result is returned over HTTP.
- Use cases **MUST NOT** select HTTP status codes or manipulate Fiber headers.
- Provider-specific values SHOULD be normalized before the use case consumes
  them.
- Constructors **MUST** reject missing required dependencies before serving
  traffic.
- Transaction ownership belongs to the use case when one business operation
  spans repositories or effects.
- Idempotency policy belongs to the application operation and durable storage,
  not only an in-memory middleware key.

## 10. Authentication and Authorization

- Authentication middleware **MUST** validate credential type, signature,
  algorithm, issuer, audience, expiry, not-before time, and revocation policy as
  applicable.
- Authorization **MUST** be checked separately for each protected operation and
  object scope.
- Parsed identity SHOULD be stored in a typed request-scoped representation.
- Handlers **MUST NOT** trust user IDs, roles, tenants, or permissions supplied
  by the request body when authenticated identity owns those values.
- Cookie settings **MUST** define `Secure`, `HttpOnly`, `SameSite`, path, domain,
  and lifetime according to deployment.
- Authentication failures SHOULD avoid revealing whether an account or resource
  exists.
- Operational annotations and API documentation are not security controls. A
  route documented as protected **MUST** have enforced middleware and tests.
- Sensitive endpoints SHOULD use route-specific rate limits and audit events.

## 11. Static Files, Redirects, and Proxies

- Static-file roots **MUST** be fixed and protected against traversal and
  dotfile access.
- Unknown static paths **MUST** preserve the intended 404 status. Do not return a
  homepage with 200 unless the application is deliberately an SPA.
- Cache immutable fingerprinted assets aggressively and HTML according to
  release freshness requirements.
- Precompressed files require correct `Accept-Encoding` negotiation,
  `Content-Encoding`, `Vary`, and MIME behavior.
- Redirect destinations derived from input **MUST** be allowlisted or constrained
  to prevent open redirects.
- Forwarded host, protocol, and client IP values **MUST** be trusted only from
  configured proxies.
- Reverse-proxy and adapter boundaries SHOULD test path prefixes, streaming,
  disconnects, headers, and error status behavior.

## 12. API Documentation and Generated Artifacts

- Runtime routes and published API documentation **MUST** agree.
- Generated OpenAPI or Swagger files **MUST** be regenerated through the
  documented command and **MUST NOT** be hand-edited.
- Concrete response schemas SHOULD describe actual envelopes rather than vague
  `object` placeholders.
- Authentication annotations **MUST** match enforced middleware.
- Error responses SHOULD document stable codes and representative shapes.
- Route, DTO, or annotation changes require a documentation consistency check.
- Generated paths SHOULD be covered by build or focused tests where stale output
  has caused drift.

## 13. Health, Readiness, Metrics, and Diagnostics

- Liveness MUST answer whether the process event loop can serve.
- Readiness MUST answer whether the instance should receive traffic.
- Startup probes SHOULD protect slow initialization without weakening liveness.
- Health handlers **MUST** be cheap, bounded, and free of secret details.
- Liveness SHOULD NOT depend on every downstream service; readiness may check
  critical initialized state with strict timeouts.
- Readiness **MUST** transition false before shutdown begins when the platform
  can route around draining instances.
- Docker, Compose, Kubernetes, and load-balancer probes **MUST** request routes
  that actually exist using tools present in the runtime image.
- Metrics endpoints SHOULD be protected by network policy or authentication
  appropriate to the environment.
- Metric labels **MUST** avoid raw paths, user IDs, query strings, and unbounded
  values. Prefer normalized route patterns.
- Profiling and debug endpoints **MUST NOT** be exposed publicly by default.

## 14. Logging, Tracing, and Observability

- Use one structured logger and write application logs to stdout or stderr in
  containerized environments.
- Access logs SHOULD include request ID, method, normalized route, status,
  latency, response size, and safe client context.
- Unknown routes SHOULD not create unbounded log-cardinality labels.
- A panic recovered by middleware **MUST** produce an observable error event with
  correlation context.
- Errors SHOULD be logged once by the boundary that owns retry, response, or
  termination decisions.
- Traces SHOULD propagate standard context through use cases and provider calls.
- Logs, traces, and metrics **MUST NOT** record authorization headers, cookies,
  tokens, secret query values, or sensitive request/response bodies.
- Observability failure **MUST NOT** break the request path unless auditing is a
  required business invariant.

## 15. Concurrency and Shared State

- Prefer immutable dependencies and request-local state.
- Shared maps, caches, and counters **MUST** be synchronized or owned by one
  goroutine.
- A mutex **MUST NOT** be added around immutable data merely to signal safety.
- Background work **MUST** have an owner, bounded queue, cancellation, failure
  handling, and shutdown join.
- Fire-and-forget goroutines from handlers are prohibited.
- Long work SHOULD move to a durable job boundary rather than outlive the HTTP
  request invisibly.
- Do not copy values containing locks, atomics, or no-copy resources.
- Prefork processes do not share ordinary memory. In-memory locks, caches, rate
  limits, and idempotency records **MUST NOT** be assumed process-wide.
- Race-sensitive changes require `go test -race` where the platform supports it.

## 16. Startup, Listening, and Graceful Shutdown

The executable entry point owns process lifecycle:

1. Load and validate configuration.
2. Construct dependencies and the Fiber app.
3. Start the listener and observe startup errors.
4. Receive platform termination signals.
5. Mark readiness false.
6. Stop accepting new work and drain in-flight requests.
7. Stop background workers and close dependencies in reverse ownership order.
8. Exit with an accurate status.

- Listen addresses and ports **MUST** be validated before startup.
- Startup failure **MUST** be logged and returned as process failure.
- SIGINT and SIGTERM SHOULD trigger graceful shutdown.
- Shutdown **MUST** be bounded and shorter than the orchestrator termination
  grace period.
- Shutdown SHOULD be idempotent.
- Reusable packages **MUST NOT** call `os.Exit` or `log.Fatal`.
- Serverless adapters **MUST** construct reusable immutable state at the
  platform-supported lifecycle boundary, not append middleware and routes on
  every request.
- Prefork, serverless, tests, and normal process serving SHOULD have separately
  verified lifecycle assumptions.

## 17. Testing Fiber Services

### Handler and app tests MUST cover

- Successful requests and exact response contracts.
- Missing, malformed, unsupported, and boundary input.
- Domain-error-to-status mapping.
- Empty collections and missing resources.
- Authentication and object-level authorization.
- Middleware order and route-group policy.
- Static, parameterized, wildcard, removed, and unknown routes.
- Health and readiness behavior.
- Panic recovery without internal leakage.
- Request context cancellation where behavior depends on it.

### Test discipline

- Construct a fresh app per test or isolated suite.
- Use Fiber's installed-version test API with bounded request timeouts.
- Close response bodies and other resources.
- Decode and compare typed response shapes; do not rely only on substring
  assertions.
- Test use cases without Fiber and repositories/providers behind their own
  contracts.
- Table-driven tests SHOULD cover equivalent validation and mapping cases.
- Tests **MUST NOT** depend on registration performed by another test.
- External services SHOULD use fakes, controlled test containers, or explicit
  integration-test targets.
- Race tests SHOULD include shared middleware state, caches, and lifecycle code.

## 18. Security Baseline

- Set deliberate request-body, header, multipart, and concurrency limits.
- Validate content types for endpoints that require a specific representation.
- Security headers SHOULD match browser behavior and embedding requirements.
- User input **MUST NOT** become filesystem paths, redirects, templates, queries,
  or commands without boundary-specific validation.
- Database access **MUST** parameterize input and enforce tenant or ownership
  scope.
- Responses SHOULD avoid unnecessary framework, server, or version disclosure.
- Denial-of-service analysis SHOULD cover expensive parsing, compression,
  regexes, pagination, aggregation, and provider fan-out.
- Dependencies and middleware SHOULD be pinned, scanned, and updated through a
  reviewed process.
- Secrets **MUST** come from an approved runtime or build-time secret mechanism,
  never source, image layers, logs, or public error bodies.

## 19. Performance and Serialization

- Measure before enabling prefork, custom serializers, aggressive compression,
  caching, or manual buffer reuse.
- Avoid converting repeatedly between strings, bytes, JSON, and domain models
  without evidence.
- Do not retain Fiber request buffers for allocation savings.
- Large responses SHOULD use pagination, bounded streaming, or asynchronous
  export according to the product contract.
- Pagination and filtering SHOULD happen at the data source when datasets can be
  large.
- Compression thresholds and algorithms SHOULD reflect response sizes and CPU
  budgets.
- Cache keys **MUST** include every value that changes representation,
  authorization, locale, or tenant scope.
- Benchmarks SHOULD represent real handler or serialization workloads and report
  allocations where useful.

## 20. Docker and Deployment

Apply the Docker companion guideline in addition to this section.

- Pin compatible Go toolchains across `go.mod`, CI, and builder images.
- Build with a frozen module graph and useful BuildKit cache mounts.
- Multi-architecture builds **MUST NOT** hard-code `GOARCH`; use build-platform
  arguments or separate targets deliberately.
- Final images SHOULD contain only the binary, certificates, timezone data, and
  required static assets.
- Run as non-root with a read-only root filesystem and dropped capabilities
  where practical.
- Container health checks **MUST** use a binary or tool that exists in the final
  image.
- Runtime image tags SHOULD be pinned according to the repository's update and
  provenance policy.
- Deployment ports, probes, termination grace, resource limits, and environment
  names **MUST** match application configuration.
- Monitoring services SHOULD not be startup dependencies of the application
  unless application correctness genuinely requires them.

## 21. Verification Workflow

Use repository commands when provided. Otherwise run the relevant subset:

```bash
gofmt -w <changed-go-files>
go test ./path/to/affected/package/...
go vet ./path/to/affected/package/...
go test ./...
go vet ./...
go build ./...
go test -race -count=1 ./...
```

Also verify, when applicable:

- Generated API documentation is current.
- The app starts with valid configuration and fails clearly with invalid
  configuration.
- Health, readiness, metrics, and unknown routes return expected statuses.
- The production container receives traffic and shuts down within its grace
  period.
- Middleware and public contracts remain compatible.

The agent **MUST** report exact commands, results, and any checks not run.

## 22. Prohibited Anti-Patterns

An agent **MUST NOT** introduce:

- A mutable package-global `*fiber.App`.
- Per-request app, middleware, or route reconstruction.
- Fiber context types in use cases or repositories.
- A goroutine that captures the Fiber request context or request-backed values.
- Silent coercion of malformed input into defaults.
- Raw internal error messages or stack traces in responses.
- HTTP 200 for every missing resource without an explicit nullable contract.
- Nil collection responses where the contract promises arrays.
- Authentication documented but not enforced.
- Permissive CORS, blanket CSRF, or public metrics without an explicit policy.
- Middleware copied without analyzing order and route scope.
- Prefork enabled as a generic performance switch.
- In-memory state assumed to be shared across prefork or replicas.
- Unbounded body sizes, pagination, concurrency, queues, retries, or provider
  fan-out.
- A health probe for a nonexistent route or a probe executable absent from the
  runtime image.
- Process termination from handlers, middleware, use cases, repositories, or
  reusable packages.
- Hand-edited generated API documentation.
- Tests that share one mutable app accidentally or normalize broken behavior
  without a documented compatibility requirement.
