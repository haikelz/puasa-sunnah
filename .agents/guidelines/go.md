# Go Coding-Agent Guideline

This document defines language-level rules for agents that create, modify, review, or verify Go code in libraries, commands, and services.

Precedence follows `general.md` §2. Where `haikel-go.md` disagrees with this guide, `haikel-go.md` wins.

## 1. Scope

- Follow the task, repository instructions, the Go version in `go.mod`, and established local conventions.
- Preserve existing public behavior and compatibility unless the task explicitly changes them.
- Make the smallest coherent change that fully solves the task. New dependencies, exported APIs, generated files, and broad refactors MUST have a clear need.
- Report any unresolved conflict between instructions rather than silently choosing a rule.

## 2. Agent Workflow

Before editing, the agent MUST:

1. Read applicable instructions, `go.mod`, relevant source and tests, `.golangci.yaml`, and configured build and generation commands.
2. Trace callers, implementations, interfaces, ownership, error behavior, and concurrency boundaries affected by the change.
3. Avoid language or library features the declared Go version does not support.
4. Reproduce a reported defect when practical and identify the narrowest correct fix.
5. Check the working tree and avoid overwriting unrelated changes.

After editing, run the verification in `haikel-go.md` §8, inspect the final diff for accidental API changes, generated artifacts, debug output, secrets, and unrelated edits, and report commands run, results, and anything not verified.

## 3. Packages and Files

- Give each package one coherent responsibility and keep package dependencies acyclic.
- Put commands in `package main`; keep reusable behavior in importable packages.
- Keep internal implementation under `internal` when outside importers must not depend on it.
- Place tests beside the code they test. Use `package_test` only when testing exported behavior alone.
- Mark platform-specific and generated files with standard suffixes, build constraints, and generated-code headers.
- AVOID catch-all packages named `util`, `common`, `helpers`, or `misc`, hidden registries, and `init` functions unless initialization cannot be explicit.

## 4. Naming and Exported APIs

- Follow Go naming conventions and standard initialisms such as `ID`, `URL`, `HTTP`, `JSON`, and `API`. Exception: domain transport types use the `Dto` prefix, as defined in `haikel-go.md` §2.
- Exported identifiers MUST have a stable reason to be public and SHOULD have doc comments that begin with the identifier and explain the contract.
- Receiver names MUST be short and consistent across a type's methods; never `this` or `self`.
- AVOID stuttering names such as `cache.CacheManager` and mutable exported package variables.
- Removing or changing exported symbols requires explicit authorization or a compatible transition path.

## 5. Constructors and Dependency Injection

Constructor shape, dependency validation, return type, and field export follow `haikel-go.md` §2 for layered Echo services and `fiber.md` for Fiber services. In every case:

- Supply required dependencies explicitly. Never introduce service locators, hidden singletons, setter injection, or package globals for convenience.
- Functional options MUST reject invalid combinations and MUST NOT hide required dependencies.

## 6. Interfaces and Receivers

- Define interfaces at the consuming boundary and keep them small. Do not create an interface solely to mock a concrete type.
- Account for typed nil values: a non-nil interface can hold a nil pointer.
- Write empty interfaces as `any`, but prefer concrete or constrained types.
- Use pointer receivers for mutation, large values, synchronization primitives, or identity. Keep receiver kind consistent for a type; never mix value and pointer receivers without a specific reason.
- Never copy a value containing `sync.Mutex`, `sync.Once`, or atomics after first use.

## 7. Context

- `context.Context` MUST be the first parameter of operations that can block, perform I/O, or be canceled.
- Pass context through call chains. Never store it in a struct, pass nil, or use it as a parameter bag.
- Use `context.Background()` only at legitimate roots. Never detach request work onto `context.Background()` to fire and forget.
- Call every derived context's cancel function, normally with `defer cancel()` right after creation.
- Context values MUST use private key types and hold only request-scoped metadata.

## 8. Errors and Panics

- Handle every error. Discarding one requires a stated reason; never use `_ =` on an error that affects correctness.
- Wrap with `%w` when crossing a boundary, using the wording rule in `haikel-go.md` §3.
- Classify with `errors.Is` and `errors.As`; never compare error strings.
- Use sentinel errors for stable conditions callers distinguish and custom error types only when callers need structured data.
- Log an error once, at the boundary that can act on it. Combine independent cleanup and primary failures with `errors.Join`.
- Never panic for expected input, configuration, cancellation, or runtime failures. Recover only at a deliberate isolation boundary and keep the failure observable.

## 9. Data, Collections, and Generics

- Use pointers to express mutation, shared identity, large values, or meaningful absence, not to avoid small copies.
- Callers MUST know whether a function retains, mutates, or copies an input slice or map. Do not expose mutable internal state without a documented contract.
- Synchronize concurrent map access with any writer. Do not rely on map iteration order.
- Go 1.22+ gives each loop iteration its own variable. Remove `v := v` copies; `copyloopvar` flags them.
- Use generics only when one implementation serves several types with a real algorithmic relationship. Prefer concrete types or small interfaces when clearer.

## 10. Concurrency and Resources

- Every goroutine MUST have an owner, a termination condition, and a way to report failure. No fire-and-forget goroutines.
- Synchronize or confine shared mutable state. Senders close channels; receivers never close channels they do not own.
- Bound worker counts, queues, retries, and parallelism. Join lifetimes with `sync.WaitGroup` or `errgroup`.
- Never sleep to coordinate goroutines or make tests pass. Never hold a lock while calling unknown code or doing slow work.
- Follow successful acquisition with `defer` cleanup when function-scoped lifetime is correct. Check cleanup errors that affect correctness, and release partially acquired resources in reverse order.

## 11. Configuration and Logging

- Parse and validate configuration once near startup into typed values. Use `time.Duration` for durations.
- Never commit, print, or log secrets, credentials, or unnecessary personal data.
- Use the repository logger with structured fields. Error level means a failed operation; warning means intentional degradation.
- Libraries return errors rather than log, unless logging is part of their contract.

## 12. Testing

- Tests MUST be deterministic, isolated, and independent of order, and cover success, boundary, and meaningful failure cases.
- Clean up resources with `t.Cleanup`. Tests that mutate globals MUST restore them and MUST NOT run in parallel with conflicting tests.
- Use table-driven tests when cases share setup. Call `t.Helper()` in helpers and `t.Parallel()` only after proving isolation.
- Prefer small fakes over interaction-heavy mocks. Inject clocks and randomness when determinism requires it.
- Never depend on external networks by default or weaken assertions to pass intermittently.
- Follow the repository's assertion style (`haikel-go.md` §8).

## 13. Formatting and Static Analysis

- The repository's `golangci-lint` formatters (`gofmt` with simplify, `goimports` with `local-prefixes`) are the source of truth. Never hand-adjust formatter output or introduce `gofumpt` unless the repository pins it.
- Format only changed files during focused work; never create a repository-wide formatting diff.
- Group imports by hand as standard library, module-local, then third-party, with one blank line between groups. Never merge groups into one block (`haikel-go.md` §2).
- Alias imports only for a collision or the role aliases in `haikel-go.md` §1. No dot imports outside narrow test conventions; blank imports need a documented side effect.
- Package names are lowercase without underscores. Preserve existing legacy names until an authorized migration.
- Struct tags MUST use the exact keys the serializer, validator, or ORM expects; formatting cannot catch a misspelled key.
- Blank-line rhythm, literal expansion, signature wrapping, and tag order follow `haikel-go.md` §2.
- Comments and identifiers use English. Remove commented-out code.
- Generated files carry the standard marker and are regenerated, never hand-edited. Linter exclusions match generated paths narrowly.
- Fix or explicitly explain every linter finding. Disable a check only at the smallest scope with a documented reason.

## 14. Verification

Use `haikel-go.md` §8 as the single command list. Additionally:

- Run `go test -race -count=1` on changed concurrent packages when the target supports it.
- Check relevant build tags, operating systems, and architectures when the change affects them.
- Distinguish failures caused by the change from pre-existing or environment-dependent failures and report both.

## 15. Prohibited Patterns

Beyond the rules above, the agent MUST NOT introduce reflection, `unsafe`, or generics where ordinary typed Go suffices; unclear boolean parameters; naked returns in nontrivial functions; process termination from reusable packages; or hand edits to generated code.
