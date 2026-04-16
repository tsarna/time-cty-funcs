# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
