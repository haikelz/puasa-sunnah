# Go Coding-Agent Guideline

This document defines language-level rules for agents that create, modify, review, or verify Go code. It is standalone and applies to libraries, commands, and services without prescribing an application architecture.

## 1. Scope and Precedence

- The agent MUST follow the task, repository instructions, supported Go version, and established local conventions.
- More specific repository instructions take precedence over this guideline. The agent MUST report any unresolved conflict rather than silently choosing a rule.
- Existing public behavior and compatibility MUST be preserved unless the task explicitly changes them.
- The agent SHOULD make the smallest coherent change that fully solves the task.
- New dependencies, exported APIs, generated files, and broad refactors MUST have a clear need.
- This guideline governs Go source, tests, package design, tooling, and build verification. It does not replace repository-specific commands.

## 2. Agent Workflow

Before editing, the agent MUST:

1. Read applicable instructions, `go.mod`, relevant source and tests, and configured build, lint, and generation commands.
2. Trace callers, implementations, interfaces, ownership, error behavior, and concurrency boundaries affected by the change.
3. Confirm the Go version and avoid language or library features unsupported by it.
4. Reproduce a reported defect when practical and identify the narrowest correct fix.
5. Check the working tree and avoid overwriting unrelated changes.

While editing, the agent MUST:

- Follow existing package boundaries and naming unless those boundaries are the problem being fixed.
- Keep behavior explicit and avoid speculative abstractions.
- Update tests and documentation that form part of the changed contract.
- Avoid unrelated formatting or cleanup.

After editing, the agent MUST:

1. Format changed Go files.
2. Run focused tests first, then the broadest practical build, test, vet, lint, and race checks.
3. Inspect the final diff for accidental API changes, generated artifacts, debug output, secrets, and unrelated edits.
4. Report commands run, results, and anything not verified.

## 3. Packages and Files

### MUST

- Give each package one coherent responsibility.
- Keep package dependencies acyclic.
- Put commands in `package main`; keep reusable behavior in importable packages.
- Keep platform-specific and generated files clearly identified through standard file suffixes, build constraints, and generated-code headers.
- Place tests beside the code they test unless external-package tests are needed to verify the public API.

### SHOULD

- Use short, lowercase, singular package names that describe what the package provides.
- Prefer cohesive files organized by responsibility over one large file or one file per trivial type.
- Use conventional lowercase file names, with underscores only where they improve clarity or implement recognized suffix patterns such as `_test.go`.
- Keep internal implementation under `internal` when outside importers must not depend on it.
- Use `package_test` when testing only exported behavior; use the package's own name when internal access is necessary.

### AVOID

- Catch-all packages named `util`, `common`, `helpers`, or `misc` without a precise capability.
- Import cycles hidden through global registries or initialization side effects.
- Files that mix unrelated concerns merely to reduce file count.
- `init` functions except where initialization cannot be expressed explicitly and ordering is unambiguous.

## 4. Naming and Exported APIs

- Names MUST follow Go conventions and use standard initialisms consistently, such as `ID`, `URL`, `HTTP`, and `JSON`.
- Exported identifiers MUST have a stable reason to be public.
- Exported packages, types, functions, methods, variables, and constants SHOULD have doc comments that begin with the identifier's name and explain the contract rather than restating syntax.
- Names SHOULD be concise in small scopes and more descriptive as scope grows.
- Receiver names MUST be short, meaningful, and consistent across a type's methods.
- Boolean names SHOULD read as conditions, such as `ready`, `hasValue`, or `canRetry`.
- Acronyms, stuttering names such as `cache.CacheManager`, and redundant type words SHOULD be avoided.
- Exported APIs MUST avoid exposing implementation details that prevent future changes.
- API changes MUST account for callers. Removing or changing exported symbols requires explicit authorization or a compatible transition path.
- Mutable exported package variables MUST be avoided. Exported constants and immutable values are preferable where a package-level value is appropriate.

## 5. Constructors and Dependency Injection

- Required dependencies MUST be supplied explicitly, normally through a constructor.
- Constructors MUST validate dependencies or configuration when invalid values would create a partially usable object.
- Constructors SHOULD return concrete types unless callers benefit from an intentionally restricted interface.
- A constructor SHOULD return an error only when construction can actually fail.
- Dependency fields SHOULD be unexported.
- Optional behavior SHOULD use clear options only when options improve readability and preserve valid defaults.
- Functional options MUST reject invalid combinations and MUST NOT hide required dependencies.
- Zero-value usability SHOULD be preferred for simple types where it is natural and safe.
- The agent MUST NOT introduce service locators, hidden singleton dependencies, or package globals for convenience.

```go
type Clock interface {
	Now() time.Time
}

type Recorder struct {
	clock Clock
}

func NewRecorder(clock Clock) (*Recorder, error) {
	if clock == nil {
		return nil, errors.New("clock is required")
	}
	return &Recorder{clock: clock}, nil
}
```

## 6. Interfaces

- Interfaces SHOULD be defined by the consuming package and describe the smallest useful capability.
- Implementations normally MUST NOT declare that they implement an interface; compile-time assertions MAY document non-obvious requirements.
- Accept interfaces and return concrete values when that makes ownership and capabilities clearer.
- An interface MUST NOT be created solely to mock a concrete type or to mirror all its methods.
- Interfaces SHOULD remain small and behavior-oriented.
- The agent MUST account for typed nil values when accepting interfaces. A non-nil interface can contain a nil pointer.
- Empty interfaces SHOULD be written as `any`, but concrete or constrained types are preferable when known.

## 7. Methods and Receivers

- Use a pointer receiver when a method mutates the receiver, the type is large, contains synchronization primitives, or identity matters.
- Use a value receiver for small immutable value-like types.
- Receiver choice MUST be consistent across a type unless there is a specific reason otherwise.
- Types containing `sync.Mutex`, `sync.Once`, atomics, or other no-copy values MUST NOT be copied after first use.
- Methods MUST document whether nil receivers are supported. Otherwise, callers and implementations SHOULD treat nil receivers as invalid.
- Fluent methods MUST make mutation versus copying unambiguous.
- Methods SHOULD not surprise callers with unrelated I/O, blocking, or global state changes.

## 8. Context

- `context.Context` MUST be the first parameter of operations that can block, perform I/O, or be canceled.
- Context MUST be passed through call chains rather than replaced without reason.
- A context MUST NOT be stored in a struct, passed as nil, or used as a general parameter bag.
- Use `context.Background()` or `context.TODO()` only at legitimate roots or while explicitly marking incomplete wiring.
- Cancellation and deadlines SHOULD be established by the caller that owns the operation's lifetime.
- Derived contexts MUST have their cancel function called, normally with `defer cancel()` immediately after creation.
- Long-running loops and expensive work SHOULD check cancellation at useful boundaries.
- Context values MUST use private key types and contain only request-scoped metadata needed across API boundaries.

## 9. Errors and Panics

### MUST

- Handle every error or explicitly justify why it is safe to discard.
- Add useful operation context with `%w` when propagating an error across a boundary.
- Use `errors.Is` and `errors.As` for classification.
- Preserve causes that callers are expected to inspect.
- Keep error strings lowercase and without trailing punctuation unless they contain a proper name or complete external message.
- Return useful zero values alongside errors unless partial results are part of the documented contract.

### SHOULD

- Use sentinel errors only for stable conditions callers need to distinguish.
- Use custom error types only when callers need structured information.
- Keep logging and error handling separate; an error should usually be logged once by the boundary that can act on it.
- Combine independent cleanup and primary failures with `errors.Join` when both matter.

### AVOID

- Comparing error strings.
- Wrapping errors with vague text such as `failed` without naming the operation.
- Logging and returning the same error at every layer.
- Panic for expected input, configuration, cancellation, or runtime failures.
- Recover except at a deliberate isolation boundary that can restore a valid state. Recovered failures MUST remain observable.

## 10. Pointers and Zero Values

- Use pointers to express mutation, shared identity, large values, or meaningful absence; do not use them merely to avoid small copies.
- The zero value SHOULD be useful when practical.
- If the zero value is invalid, constructors or validation MUST make that clear.
- Nil and empty collections MUST be treated consistently according to the API contract.
- Optional scalar values MAY use pointers when absence differs from the scalar's zero value; a small option type MAY be clearer in domain-neutral libraries.
- Returned pointers MUST NOT expose mutable internal state unless that sharing is intentional and documented.
- Do not take pointers to loop variables whose storage or lifetime does not represent the intended element.
- Copy methods MUST define whether nested maps, slices, and pointers are shallow- or deep-copied.

## 11. Slices and Maps

- Callers MUST know whether a function retains, mutates, or copies an input slice or map.
- Functions SHOULD avoid retaining large backing arrays through small subslices; copy when retention would be costly.
- Preallocate with a known or well-estimated capacity when it materially reduces allocation and does not obscure code.
- Map reads from a nil map are valid; writes require initialization.
- Concurrent map access with any writer MUST be synchronized or ownership-confined.
- Use the comma-ok form when absence differs from a stored zero value.
- Do not rely on map iteration order. Sort keys when deterministic output is required.
- Be careful when deleting or appending during iteration; behavior MUST be intentional and tested.
- APIs MUST document whether nil and empty slices produce different serialized or observable results.

## 12. Generics

- Generics SHOULD be used only when one implementation clearly serves multiple types while preserving type safety.
- Type parameters MUST express a real algorithmic relationship, not hide unrelated behavior.
- Constraints SHOULD be as narrow as required and use standard constraints or type sets where possible.
- Prefer ordinary functions, concrete types, or small interfaces when they are clearer.
- Generic APIs MUST remain readable at call sites and produce understandable compiler errors.
- Do not use reflection where a simple generic implementation is safer, and do not use generics where straightforward duplication is clearer.
- Zero values of type parameters MUST be handled deliberately.
- Type assertions from `any` inside generic code SHOULD be avoided because they defeat static guarantees.

## 13. Concurrency

### MUST

- Every goroutine must have an owner, a termination condition, and a way to observe or handle failure.
- Shared mutable state must be synchronized or confined to one goroutine.
- Channel ownership and closure responsibility must be clear. Senders normally close channels; receivers MUST NOT close channels they do not own.
- Blocking sends, receives, and lock acquisition paths must account for cancellation or bounded lifetime where needed.
- Concurrent code must avoid goroutine leaks on early returns and errors.

### SHOULD

- Prefer synchronous code until concurrency provides a demonstrated benefit.
- Use `sync.WaitGroup`, `errgroup` when already available and appropriate, or an explicit equivalent to join worker lifetimes.
- Bound worker counts, queues, retries, and parallelism.
- Keep critical sections small and never copy a mutex after use.
- Define ordering and backpressure semantics explicitly.
- Test concurrency-sensitive code with the race detector and repeated runs.

### AVOID

- Fire-and-forget goroutines.
- Sleeping to coordinate goroutines or make tests pass.
- Holding locks while calling unknown code or performing slow work.
- Closing a channel from multiple possible paths without synchronization.
- Using channels where a mutex or direct call is simpler.

## 14. Resource Ownership and Cleanup

- The code that acquires a resource MUST make ownership and release responsibility explicit.
- Successful acquisition SHOULD be followed immediately by `defer` cleanup when function-scoped lifetime is correct.
- Cleanup errors MUST be checked when they can affect correctness.
- Streams, files, responses, timers, tickers, subprocesses, and goroutines MUST be stopped, closed, waited for, or transferred to a documented owner.
- `defer` inside an unbounded loop SHOULD be avoided; use a helper scope or explicit cleanup.
- A function returning a resource MUST document how the caller releases it.
- Partial initialization MUST clean up already-acquired resources in reverse order.
- Shutdown SHOULD be idempotent and bounded by context when it can block.

## 15. Configuration

- Configuration MUST be parsed and validated near program startup or package construction.
- Runtime code SHOULD receive typed configuration rather than repeatedly reading process-global state.
- Required values, ranges, units, and interactions MUST be validated before work starts.
- Durations SHOULD use `time.Duration`; byte sizes and counts SHOULD use types and names that make units clear.
- Defaults MUST be safe, documented, and distinguishable from required values.
- Sensitive values MUST NOT be committed, printed, included in errors, or exposed through diagnostic output.
- Tests SHOULD construct configuration directly and avoid dependence on the developer's environment.
- Package initialization MUST NOT terminate the process because configuration is absent.

## 16. Logging

- Use the repository's established logger; do not introduce another logging abstraction without need.
- Logs SHOULD be structured and include stable operation names and relevant identifiers.
- Log levels MUST reflect actionability: errors for failed operations, warnings for recoverable abnormal conditions, and debug information for diagnostics.
- Secrets, credentials, private content, and unnecessary personal data MUST NOT be logged.
- Errors SHOULD be logged at the boundary that owns retry, response, or termination decisions.
- Libraries SHOULD return errors rather than write logs unless logging is an explicit part of their contract.
- Logging MUST NOT change control flow or be required for correctness.
- Hot paths SHOULD avoid expensive formatting and excessive cardinality.

## 17. Testing

### MUST

- Tests must be deterministic, isolated, and independent of execution order.
- New or changed behavior must have focused coverage for success, boundary conditions, and meaningful failures.
- Test failures must identify the case and expected behavior clearly.
- Resources and goroutines started by tests must be cleaned up, preferably with `t.Cleanup`.
- Tests that mutate process-global state must restore it and must not run concurrently with conflicting tests.

### SHOULD

- Use table-driven tests when several cases share setup and assertions.
- Use subtests with descriptive names.
- Call `t.Helper()` in reusable test helpers and `t.Parallel()` only after proving isolation.
- Prefer small fakes or stubs over interaction-heavy mocks.
- Test public behavior rather than implementation details.
- Inject clocks, randomness, and side-effecting dependencies when determinism requires control.
- Use fuzz tests for parsers, codecs, and invariant-heavy inputs where useful.
- Use benchmarks only for meaningful performance questions, with allocation reporting when relevant.

Tests MUST NOT use arbitrary sleeps as synchronization, depend on external networks by default, or weaken assertions merely to pass intermittently.

## 18. Formatting and Static Analysis

- All changed Go files MUST be formatted with the repository's configured formatter; otherwise use `gofmt`.
- Imports SHOULD be managed with the repository's configured import tool or conventional Go tooling.
- The agent MUST run `go vet` on affected packages and SHOULD run the configured linter.
- Linter findings MUST be fixed or explicitly explained; disabling a check requires a narrow, documented reason.
- Generated files MUST be regenerated through their declared generator, not hand-edited.
- Build constraints MUST use current syntax and remain valid for all intended targets.
- Comments and identifiers MUST use English unless a public protocol requires exact text.

Typical commands are:

```bash
gofmt -w <changed-go-files>
go test ./path/to/affected/package/...
go vet ./path/to/affected/package/...
```

## 19. Build, Test, and Race Verification

Use repository commands when provided. Otherwise, from the module or workspace root, run the broadest practical subset:

```bash
go test ./...
go vet ./...
go build ./...
go test -race -count=1 ./...
```

- Focused tests SHOULD run before the full suite for faster diagnosis.
- Race verification MUST run for changed concurrent code when the target supports it.
- Relevant build tags, operating systems, architectures, and command packages SHOULD be checked when the change affects them.
- Cached success SHOULD be bypassed with `-count=1` when investigating nondeterminism or verifying concurrency.
- The agent MUST distinguish failures caused by the change from pre-existing or environment-dependent failures and report both accurately.
- The final diff SHOULD be checked for whitespace errors and unintended generated changes.

## 20. Prohibited Anti-Patterns

The agent MUST NOT introduce:

- Ignored errors, blank-identifier error suppression, or unchecked cleanup where failure matters.
- Error-string matching or panic for expected failures.
- Hidden dependencies through mutable package globals, service locators, or surprising `init` behavior.
- Oversized consumer interfaces or interfaces created only for mocking.
- Context stored in structs, nil contexts, or context values used for ordinary parameters.
- Unbounded goroutines, queues, retries, recursion, or parallelism.
- Concurrent map writes, unsynchronized mutable state, copied locks, or ambiguous channel closure.
- Arbitrary sleeps for synchronization.
- Exposed mutable internal slices, maps, or pointers without an explicit contract.
- Reflection, `unsafe`, or generics where ordinary typed Go is clear and sufficient.
- Premature abstraction, speculative extensibility, or needless dependency additions.
- Boolean parameters whose meaning is unclear at call sites; use separate methods or an explicit option type.
- Naked returns in nontrivial functions, deeply nested control flow, or clever code that obscures ownership and errors.
- Process termination from reusable packages.
- Logging of secrets or duplicate logging at every layer.
- Hand edits to generated code or unrelated repository-wide formatting.
- Tests that depend on timing, external state, global randomness, or execution order.
