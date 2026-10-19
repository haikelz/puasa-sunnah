# Haikel Go Engineering Profile

This profile records the owner's demonstrated Go service conventions. Load it
with `preferences.md` and `go.md`, then add the applicable Echo,
GORM/PostgreSQL, Docker, and infrastructure guidelines. Repository-local rules,
finance documentation, and public contracts take precedence.

## 1. Service Shape and Ownership

The preferred service flow is explicit and domain-oriented:

```text
route → HTTP handler → domain usecase → repository interface → GORM/PostgreSQL
```

- Put domain request/filter/response DTOs and interfaces in the domain root.
- Keep HTTP parsing, transport validation, identity extraction, and response
  envelopes in `delivery/http` handlers.
- Keep business invariants, cross-repository orchestration, and provider
  coordination in usecases.
- Keep GORM, SQL, and query construction in repositories.
- Wire dependencies centrally at application startup. Handlers MUST NOT create
  repositories or reach package-global services directly.
- Follow the existing domain boundary for compatible changes. Do not introduce a
  new technical layer, generic repository, or service locator without a proven
  need.

For a layered domain, prefer this file placement unless the repository already
has a stronger convention:

```text
internal/domains/<domain>/
├── dto.go
├── error.go
├── repository.go
├── usecase.go
├── delivery/http/<domain>_handler.go
├── repository/<domain>_repository.go
└── usecase/<domain>_usecase.go
```

- Keep transport DTOs for one domain together when they remain cohesive. Split
  the file by capability only after it develops distinct responsibilities.
- Keep stable domain errors and state helpers in the domain root, not in the
  delivery or persistence implementation.
- Define the narrow `Repository` and `Usecase` contracts at the domain boundary;
  put concrete implementations in their role directories.
- Name concrete types by domain and role, such as `OrderHandler`,
  `OrderUsecase`, and `OrderRepository`. Keep receiver names consistent: `h`,
  `u`, and `r` respectively.
- In a long interface, group related methods with one short section comment and
  one blank line. Do not add a heading before every method.

## 2. Formatting, Imports, and Naming

- Target the Go version declared by `go.mod`.
- Run `gofmt -s` for every changed Go file. When the project uses `goimports`,
  run it without `-local` for this import order, unless repository tooling
  explicitly requires another order; inspect the resulting groups afterward.
  Formatter output is authoritative for indentation and alignment.
- Treat whitespace as part of readability. The formatter owns indentation and
  alignment; the developer still owns intentional blank lines between semantic
  phases.
- Keep import groups as standard library, module-local, then third-party. Use one
  blank line between groups and no blank lines within a group. Sort imports
  within each group; do not use `goimports -local <module-path>` to enforce this
  profile, because that flag moves module-local imports *after* third-party
  imports. Follow repository-enforced import tooling if it requires otherwise.
- Alias an import only to resolve a collision or make a non-obvious package
  identity explicit. Keep an established explicit alias consistent throughout a
  package.
- Use standard Go initialisms: `ID`, `URL`, `HTTP`, `JSON`, `API`, and `DTO`.
- Use concise, lowercase package names; preserve existing legacy names instead
  of renaming a package as incidental cleanup.
- Use DTO suffixes for transport contracts and `Usecase`, `Repository`, and
  `Handler` suffixes where the architecture already makes that ownership useful.
- Constructors should make dependencies explicit and normally return the domain
  interface when callers should depend only on the capability. Use a keyed,
  multiline literal so dependency mapping remains visible after formatting.
- Declare one named struct field per line. Do not compress fields of the same type
  into `first, second string` when the struct represents a domain, transport,
  persistence, or dependency contract.
- Do not use positional literals for application structs, results, cursors,
  errors, or test fixtures. Keyed fields protect meaning when fields are added or
  reordered. Compact positional forms remain acceptable only for conventional
  value types where the meaning is unmistakable.

```go
func NewBannerUsecase(repo banner.Repository) banner.Usecase {
	return &bannerUsecase{
		repository: repo,
	}
}
```

### Visual rhythm and semantic spacing

The owner prefers code with deliberate breathing room. Do not compress a
function merely because `gofmt` accepts it. Readability comes from visible
groups of related statements, not from line count.

- Use one blank line between distinct phases. Common phases are authentication,
  input parsing, normalization, validation, dependency calls, error handling,
  result transformation, and response construction.
- Keep an operation adjacent to the error check that handles it. Put the blank
  line after the completed error block, before the next phase.
- After a terminating guard clause, add a blank line before normal-path work
  when the next statement begins a different concern.
- Keep consecutive checks together when they validate the same input or
  invariant. Do not insert a blank line after every `if` mechanically.
- In repositories, visually separate transaction setup, query execution,
  scan/error classification, domain mapping, audit or outbox work, commit, and
  the final return.
- In loops, separate row-local declarations, scanning, error handling,
  transformation, and append/update work when several of those phases exist.
- In tests, use arrange, act, and assert groups. Keep the setup for one scenario
  together rather than scattering blank lines through every assignment.
- Use multiline keyed literals for constructors and application structs. Expand
  callbacks with meaningful work instead of hiding their body on one line.
- Wrap calls and argument lists when they mix several domain values, contain a
  nested literal, or require horizontal rereading to understand ownership.
- Use exactly one blank line for a boundary. Never add repeated blank lines,
  blank lines immediately inside braces, or whitespace that separates an
  operation from its error check.
- Keep one blank line between top-level types, constructors, and methods. Keep
  related sentinel errors in one `var` block instead of spacing every error into
  a separate declaration.
- Use guard clauses to keep the successful workflow at the lowest indentation.
  Do not wrap the remainder of a function in `else` after a branch returns.
- For a multiline boolean expression, leave `&&` or `||` on the preceding line,
  use one condition per line, and let `gofmt` determine continuation indentation.
- For a multiline GORM chain, put one operation per line and leave the period at
  the end of the preceding line. Keep the terminal operation and `.Error`
  visible at the end of the chain.
- Keep route registration as one contiguous block. Add one blank line only when
  a separate route group marks a different access boundary.
- Keep a request declaration, bind, and validation checks adjacent because they
  are one input phase. Start a new visual group only after that phase succeeds.
- Keep a simple delegation as a direct return. Do not introduce a temporary
  result solely to manufacture spacing; reserve phase spacing for work that has
  more than one concern.
- Place a parsed path or query value beside its validation. Add a blank line
  before binding the body or invoking a usecase, not between the parse and its
  error check.
- Group dependency guards together, then separate them from domain checks and
  I/O. Keep adjacent guards for the same invariant together; do not insert a
  blank line simply because an `if` returned.
- For short repository methods, keep the query, its error check, and the final
  return together when they form one lookup. Add a blank line before the return
  only when mapping or another distinct phase follows the check.

Preferred handler rhythm:

```go
func (h *Handler) Update(c echo.Context) error {
	actor, err := h.authenticate(c)
	if err != nil {
		return respondError(c, err)
	}

	var input UpdateInput
	if err := c.Bind(&input); err != nil {
		return respondError(c, ErrInvalidInput)
	}
	if err := c.Validate(&input); err != nil {
		return respondError(c, err)
	}

	result, err := h.usecase.Update(c.Request().Context(), actor, input)
	if err != nil {
		return respondError(c, err)
	}

	return respondSuccess(c, result)
}
```

Preferred usecase rhythm (the blank lines separate different concerns, not
every error check):

```go
func (u *orderUsecase) Assign(ctx context.Context, orderID, driverID uuid.UUID) error {
	if u.repository == nil {
		return ErrRepositoryUnavailable
	}
	if driverID == uuid.Nil {
		return ErrInvalidDriver
	}

	order, err := u.repository.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != StatusSearching {
		return ErrInvalidStateTransition
	}

	return u.repository.AssignDriver(ctx, orderID, driverID)
}
```

Before finishing a broad cleanup, inventory and inspect every handwritten Go
file in the declared scope after formatting. Include handlers, usecases,
repositories, workers, commands, and tests; exclude generated files explicitly.
Passing `gofmt` alone does not prove that their visual rhythm matches this
profile.

## 3. Handler and Usecase Workflows

### HTTP boundaries and validation

Handlers should execute one consistent sequence:

1. Authenticate and reject disallowed roles when the route requires it.
2. Read and parse route, query, form, or body input.
3. Bind into a domain DTO and run structural validation.
4. Add server-derived actor or ownership data.
5. Call one usecase operation.
6. Translate the result into the established success or error envelope.

```go
request := &payment.DtoCoinPayment{}
if err := c.Bind(request); err != nil {
	return models.ErrorResponse(c, err.Error())
}
if err := c.Validate(request); err != nil {
	return models.ErrorResponse(c, helper.MappingError(err))
}

result, err := h.paymentUsecase.PayWithCoin(c, request)
if err != nil {
	return models.ErrorResponse(c, err.Error())
}

return models.SuccessResponse(c, result, "Payment successful")
```

- DTO tags MUST match their input boundary: `json`, `form`, `query`, and
  `validate` serve different purposes.
- The handler MUST reject invalid bind/validation input before calling a usecase.
- Server-derived actor IDs, role IDs, and audit fields MUST NOT come from a
  request body.
- Database-dependent rules, ownership, financial thresholds, and state
  transitions belong in usecases, not only handlers.
- Preserve existing route paths, status behavior, response envelope fields, and
  Swagger annotations unless an API contract change is explicit.
- Regenerate Swagger from handlers; never hand-edit generated documentation.

Keep bind and validation checks adjacent because they form one input phase. Add
a blank line only after that phase is complete. Do not move database lookups,
state transitions, transaction control, or provider orchestration into a handler
to avoid adding a usecase method.

### Usecase workflow

Usecases should read as a top-to-bottom business procedure:

1. Guard required dependencies and invalid initial state.
2. Load the actor and domain records needed by the rule.
3. Validate ownership, limits, and allowed state transitions.
4. Calculate or construct the intended domain change.
5. Persist through repository contracts or one owned transaction.
6. Perform post-commit cache, notification, or provider work according to its
   delivery guarantee.
7. Return the domain result.

- Put one blank line between these phases, not between every assignment.
- Keep related nil-dependency guards together at the start of the function.
- Build nontrivial request, model, and response values with multiline keyed
  literals. Field order should follow the destination contract or a stable
  domain grouping.
- Use immediate error returns. Add operation context when crossing a boundary;
  do not log and return the same error from every layer.
- Extract a helper when it owns a coherent repeated workflow or materially
  reduces nesting. Do not create a helper merely to shorten a long function.
- Keep external side effects after a successful database commit unless an
  outbox or another durable design intentionally couples their delivery.

## 4. Context, GORM, and Repository Work

- Pass the request context through blocking work. In Echo/GORM code, use
  `db.WithContext(c.Request().Context())` for new and touched queries.
- Use parameterized GORM predicates. Apply a deterministic ordering before
  offset/limit pagination.
- Keep count and list filters identical. Preserve soft-delete semantics in joins
  and raw SQL explicitly.
- Use `errors.Is(err, gorm.ErrRecordNotFound)` for missing records; never match
  error strings.
- Preserve table key types. Do not casually interchange `UUID`, `uint`, and
  `uint64` identifiers.
- Use exact integer currency units, never `float32` or `float64`, for money.
- Use explicit update maps/columns for state transitions. Use `gorm.Expr` or a
  guarded update predicate for atomic counter and balance changes.
- Check rows affected when an update depends on an expected prior state.

Repository methods should expose this visual order:

1. Declare the destination value.
2. Build and execute the query.
3. Classify or return the query error immediately.
4. Map and return the result.

Format fluent queries vertically once they contain several operations:

```go
var order models.Order
err := r.db.WithContext(ctx).
	Preload("Customer").
	Preload("History", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).
	Where("id = ?", id).
	First(&order).Error
if err != nil {
	return nil, err
}
return &order, nil
```

For state transitions, prefer a transaction callback with the following visible
phases:

1. Lock or load the current row.
2. Return successfully for an idempotent terminal state when that is the
   contract.
3. Reject invalid state and ownership.
4. Apply the update.
5. Insert history, ledger, audit, or outbox records.
6. Return from the callback so GORM owns commit or rollback.
7. Update best-effort cache state only after commit.

```go
err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	var order models.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&order).Error; err != nil {
		return err
	}

	if order.Status == StatusCompleted {
		return nil
	}
	if order.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}

	if err := tx.Model(&order).
		Update("status", StatusCompleted).Error; err != nil {
		return err
	}

	history := models.OrderHistory{
		OrderID: order.ID,
		Status:  StatusCompleted,
	}

	return tx.Create(&history).Error
})
if err != nil {
	return fmt.Errorf("complete order: %w", err)
}

return nil
```

Do not interleave post-commit cache or notification work inside the transaction
callback. Do not use manual `Begin`, repeated `Rollback`, and `Commit` when a
transaction callback expresses the same ownership more safely.

## 5. Finance and Transaction Invariants

Financial behavior is correctness-critical. One money mutation and every
corresponding ledger, history, log, or journal entry MUST commit or roll back in
the same PostgreSQL transaction.

This rule applies to payments, callbacks, wallets, top-ups, settlement,
commissions, withdrawals, refunds, bank accounts, and journals.

```go
return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := repository.UpdateBalanceTx(tx, walletID, amount); err != nil {
        return fmt.Errorf("update wallet balance: %w", err)
    }
    if err := repository.CreateLedgerTx(tx, ledger); err != nil {
        return fmt.Errorf("create wallet ledger: %w", err)
    }
    return nil
})
```

- Pass the exact outer `*gorm.DB` to every transaction-aware repository method.
  Never call a repository method that opens a separate base-connection
  transaction from inside an outer financial transaction.
- Return every error from the transaction callback to guarantee rollback.
- Callbacks and retryable operations MUST be idempotent. Use stable, unique event
  keys or an equivalent durable deduplication rule.
- Do not perform irreversible external side effects inside a database transaction
  unless a durable outbox/idempotency design makes retry behavior safe.
- Ownership and chart-of-account semantics are distinct. Never model an
  accounting account as a pseudo-wallet owner simply to simplify a query.

## 6. Authentication, Errors, and Logging

- JWT verification authenticates the request; authorization still requires role
  and resource-ownership checks.
- Treat missing or malformed claims as unauthorized. Do not use unchecked type
  assertions for security-sensitive claim values.
- Wrap operational errors with context and `%w`; use `errors.Is`/`errors.As` for
  classification.
- Preserve established public error envelopes for compatibility. Do not turn a
  focused change into an unrequested API-wide status-code migration.
- Log at the boundary that owns the response, retry, or operation failure. Use
  structured fields for stable identifiers, operation, state, and provider.
- Never log secrets, access tokens, raw payment callbacks, credentials, or full
  personally identifiable payloads.
- Use `Warn` for intentional best-effort degradation and `Error` for a failed
  operation. Never suppress an error silently unless the contract explicitly
  permits it and the decision is observable.

## 7. Configuration, Migrations, and Runtime

- Parse configuration once through the established configuration boundary;
  validate required production settings before dependent work begins.
- Add new settings to the typed config and example environment file. Never put
  real credentials in examples, logs, tests, source, or generated artifacts.
- PostgreSQL schema changes MUST use the approved migration command and ordered
  migration registry. Application startup MUST NOT modify production schema.
- Prefer backward-compatible expand/contract migrations and test rerun/failure
  behavior. Build the migration command when migration code changes.
- Preserve startup, shutdown, health-check, timeout, middleware, and deployment
  invariants unless the task explicitly changes them.
- New dependencies should be injected. Avoid extending package-global state,
  especially for provider clients, logging, and persistent connections.

## 8. Tests and Proof

- Use focused, table-driven tests for rule matrices and exact financial
  boundaries: below, at, and above a threshold.
- Prefer small fakes that satisfy the domain interface for usecase tests. Assert
  result, error, and meaningful side effects/call count.
- Handler tests should construct an Echo context and stub the usecase boundary.
- Finance, callback, and state-transition changes MUST test rollback,
  idempotency, duplicate events, prior-state guards, ownership, and ledger
  consistency.
- Tests MUST not invoke live Firebase, S3, Redis, Kafka, payment providers, or
  customer endpoints. Restore package globals after tests that alter them and do
  not run conflicting global-state tests in parallel.
- Run the repository formatter, focused tests, `go test ./...`, `go vet ./...`,
  and the relevant command build. Do not rely on stale Makefile targets without
  verifying they point at the active command.

## 9. Legacy Patterns to Avoid Extending

- Do not reproduce nested base-connection transactions in a financial usecase;
  they can commit independently from the intended outer transaction.
- Do not write raw provider callback payloads to local files. Prefer redacted,
  structured audit data under a reviewed retention policy.
- Do not assume all existing HTTP errors use ideal status codes; preserve the
  contract locally while keeping new versioned APIs precise.
- Do not extend unchecked JWT claim assertions, Echo context leakage into new
  infrastructure, ignored errors, manual timestamps without need, or temporary
  environment-specific response rewriting.
- Do not depend on cron execution from every API replica or assume a configured
  Kafka topic implies an active consumer. Verify operational ownership first.
