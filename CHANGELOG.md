# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-07-14

### Added

- **`Externs()` — the real signatures of the functions cty cannot describe, for functy
  hosts.** `externs.cty` (embedded; exposed as opaque bytes via `Externs()` and
  `ExternsFilename`) declares 12 of the 31 functions this package provides. This package
  does **not** import functy — the bytes are opaque here, and `embed` is stdlib. A functy
  host registers them with `parser.RegisterExterns(timecty.Externs(), timecty.ExternsFilename)`.

  Two kinds of function are misrepresented by their own cty metadata:

  - **Faked optionals.** cty can only make a function's *trailing* parameters optional, so
    an optional argument has to be a variadic — which erases its name, its type, and its
    default. `fromunix(n, unit = "s")` reflects as `fromunix(n, ...args)`. Declared:
    `now`, `strptime`, `fromunix`, `unix`, `timezone`, `formatduration`, `nextzoneserial`.
  - **Genuine overloads.** A cty function has one signature. `parsetime(s)` reads a
    timestamp while `parsetime(format, s)` reads a *format* and then a timestamp;
    `duration` takes a string *or* a number and a unit; and `timeadd`/`timesub` have a
    **return type that depends on their arguments** — something cty cannot express even in
    principle. Each is declared as a set of forms, one per shape, each with its own return
    type.

  The other 19 functions are deliberately **not** declared: their cty metadata is complete
  (fixed arity, concrete parameter types, a static return type), so an extern would only be
  a second place for the same facts to drift. A test enforces the split in both directions.

### Changed

- **Every function now carries a cty `Description`, and every function without an extern
  documents its parameters.** Previously there was not one `Description` in the package —
  so any reflection over the cty metadata reported functions that exist but are
  undocumented.
- **Parameter types are declared honestly where cty allows it.** `timezone`'s and
  `nextzoneserial`'s optional time, `duration`'s unit, and `timesub`'s first argument were
  all declared `DynamicPseudoType` and then type-checked by hand inside the `Type` func;
  they now say what they accept, and cty rejects the rest. The parameters that remain
  dynamic are the genuine unions (`timesub`'s second argument, `duration`'s first, the DNS
  serial) — cty has no union type, which is why those functions are declared in
  `externs.cty` as one form per type.

### Fixed

- **Surplus arguments are now rejected.** A cty `VarParam` has no upper bound of its own,
  so `now`, `fromunix`, `unix` and `formatduration` — which use one to fake an *optional*
  argument — accepted any number of trailing arguments and silently dropped them:
  `now("UTC", "junk")` returned a time, and `fromunix(0, "s", "x")` succeeded. Each now
  declares its ceiling, as `parsetime`, `strptime`, `timezone` and `duration` already did.

  This rejects calls that previously "worked", but only ones that were already nonsense —
  the surplus arguments never had any effect.

## [0.2.0] - 2026-04-16

### Added

- [rich-cty-types](https://github.com/tsarna/rich-cty-types) integration: the
  `time` and `duration` capsule types now implement the `Stringable` and
  `Gettable` interfaces. When `richcty.GetGenericFunctions()` is registered in
  the eval context alongside `GetTimeFunctions()`, HCL expressions can use
  `tostring(t)`, `tostring(d)`, `get(t, part)`, and `get(d, unit)` generically.
  - `tostring(t)` formats a `time` as RFC 3339 with nanosecond precision
    (equivalent to `formattime("@rfc3339nano", t)`).
  - `tostring(d)` formats a `duration` using Go syntax (equivalent to
    `formatduration(d)`).
  - `get(t, part)` and `get(d, unit)` take the same part/unit names as the
    removed `timepart`/`durationpart`.
- New exported wrapper types `Timestamp` and `Duration` (embedding `time.Time`
  and `time.Duration` respectively) carry the `ToString` and `Get` method
  implementations.

### Changed

- **Breaking:** The underlying Go type of `TimeCapsuleType` changed from
  `time.Time` to `Timestamp`, and `DurationCapsuleType` from `time.Duration` to
  `Duration`. This is necessary because the `Stringable` / `Gettable`
  interfaces cannot be attached to stdlib types. The public helpers
  `NewTimeCapsule`, `GetTime`, `NewDurationCapsule`, and `GetDuration` keep
  their signatures and handle the wrapping transparently. Only code that
  reaches for `val.EncapsulatedValue()` directly and type-asserts to
  `*time.Time` / `*time.Duration` needs to update to `*Timestamp` / `*Duration`
  (and read the embedded `Time` / `Duration` field).

### Removed

- **Breaking:** `TimePartFunc` / the `timepart(t, part)` function has been
  removed. Use the rich-cty-types generic `get(t, part)` instead.
- **Breaking:** `DurationPartFunc` / the `durationpart(d, unit)` function has
  been removed. Use the rich-cty-types generic `get(d, unit)` instead.

## [0.1.1] - 2026-04-07

### Changed

- Bump `github.com/sosodev/duration` to v1.4.0.
- Bump `github.com/itchyny/timefmt-go` to v0.1.8.

### Added

- Renovate configuration for automated dependency updates.

## [0.1.0]

Initial release.

[Unreleased]: https://github.com/tsarna/time-cty-funcs/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/tsarna/time-cty-funcs/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/tsarna/time-cty-funcs/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/tsarna/time-cty-funcs/releases/tag/v0.1.0
