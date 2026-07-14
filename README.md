# time-cty-funcs

cty functions and types for dealing with time; mainly used in HCL2 templates.

[![CI](https://github.com/tsarna/time-cty-funcs/actions/workflows/ci.yml/badge.svg)](https://github.com/tsarna/time-cty-funcs/actions/workflows/ci.yml)

## Overview

This package provides two [go-cty](https://github.com/zclconf/go-cty) capsule types — `time` and `duration` — plus a comprehensive set of functions for working with them in HCL2 expression evaluation contexts.

## Types

### `timecty.TimeCapsuleType`

A cty capsule type wrapping Go's `time.Time`. Supports equality (`==`, `!=`) via `CapsuleOps`. Timezone is stored inside the value; comparison is always by absolute UTC instant regardless of stored timezone.

### `timecty.DurationCapsuleType`

A cty capsule type wrapping Go's `time.Duration` (int64 nanoseconds; range ±~292 years). Supports equality (`==`, `!=`) via `CapsuleOps`. Use `duration::lt`/`duration::gt` (or extract via `get(d, unit)` and compare numerically) for ordering.

**Limitation:** Go's `time.Duration` cannot represent calendar months or years exactly. ISO 8601 durations like `P1Y` or `P1M` are rejected; use `time::add_years()` / `time::add_months()` instead.

### Helper functions

```go
timecty.NewTimeCapsule(t time.Time) cty.Value
timecty.GetTime(val cty.Value) (time.Time, error)
timecty.NewDurationCapsule(d time.Duration) cty.Value
timecty.GetDuration(val cty.Value) (time.Duration, error)
```

## Registration

```go
import timecty "github.com/tsarna/time-cty-funcs"

// Add all time functions to your eval context:
for name, fn := range timecty.GetTimeFunctions() {
    funcs[name] = fn
}
```

`GetTimeFunctions()` returns the functions described below, keyed by the name they are callable under. The names are **namespaced** — `time::add`, `duration::truncate`, `dns::next_zone_serial` — following HashiCorp's conventions for provider-defined functions: the leaf name does not repeat the namespace, and namespaced functions use underscores (HCL's *built-in* functions run words together for historical reasons).

`duration` keeps a bare name: it is the type constructor, and reads as one.

There is deliberately no `timeadd`. That name belongs to go-cty's own `stdlib.TimeAddFunc`, the string-in/string-out function HCL has always had; a host that wants it registers it directly. `time::add` carries no compatibility burden as a result, which is why every one of its forms returns a `time`.

### functy externs — the signatures cty cannot express

Some of these functions cannot describe themselves through cty's metadata. cty can only make a function's *trailing* parameters optional, so an optional argument has to be faked with a variadic — which erases its name, its type, and its default: `time::from_unix(n, unit = "s")` reflects as `from_unix(n, ...args)`. And a cty function has one signature, so a function whose argument shapes differ per call (`time::parse(s)` vs `time::parse(format, s)`), or whose parameters are a union (`dns::parse_zone_serial` takes a number *or* a string), cannot be described at all.

`Externs()` returns [functy](https://github.com/tsarna/functy) declarations stating what each really accepts, keyed by a filename to report in diagnostics. A functy host registers them so that `help()`, generated documentation, and editor tooling show the true signature:

```go
for name, src := range timecty.Externs() {
    parser.RegisterExterns(src, name)
}
```

```text
> help("time::parse")
time::parse(s: string) -> time
time::parse(format: string, s: string) -> time
time::parse(format: string, s: string, tz: string) -> time

Parse a timestamp.
```

This package does **not** import functy — the bytes are opaque here, and `embed` is stdlib. Functions absent from the declarations are absent deliberately: their cty metadata is complete, so an extern would only be a second place for the same facts to drift.

### rich-cty-types integration

The `time` and `duration` capsule types implement the [rich-cty-types](https://github.com/tsarna/rich-cty-types) `Stringable` and `Gettable` interfaces. To expose the generic `tostring` and `get` functions in your eval context, merge them in:

```go
import (
    timecty "github.com/tsarna/time-cty-funcs"
    richcty "github.com/tsarna/rich-cty-types"
)

funcs := richcty.GetGenericFunctions()       // tostring, get, length, ...
for name, fn := range timecty.GetTimeFunctions() {
    funcs[name] = fn
}
```

With these registered:

- `tostring(t)` formats a `time` as RFC 3339 with nanosecond precision (equivalent to `time::format("@rfc3339nano", t)`).
- `tostring(d)` formats a `duration` using Go syntax (equivalent to `duration::format(d)`).
- `get(t, part)` extracts a calendar field from a `time`. Valid `part` values: `"year"`, `"month"`, `"day"`, `"hour"`, `"minute"`, `"second"`, `"nanosecond"`, `"weekday"` (0=Sunday), `"yearday"`, `"isoweek"`, `"isoyear"`.
- `get(d, unit)` extracts a `duration` in the given unit. `"h"`, `"m"`, `"s"` return floats; `"ms"`, `"us"`, `"ns"` return integers.

The part/unit accessors are available **only** through `get()`; the previous `timepart()` and `durationpart()` functions have been removed.

## String Formats

### Timestamps — ISO 8601 / RFC 3339

```
2024-01-15T10:30:00Z                  # UTC
2024-01-15T10:30:00+05:30             # With offset
2024-01-15T10:30:00.123456789Z        # Sub-second precision
```

### Durations — ISO 8601 P-notation

```
PT5M           # 5 minutes
PT1H30M        # 1 hour 30 minutes
P1DT12H        # 1 day 12 hours (= 36h fixed)
PT0.5S         # 500 milliseconds
```

### Durations — Go format

```
5m             # 5 minutes
1h30m          # 1 hour 30 minutes
500ms          # 500 milliseconds
```

### Named format aliases (`@` prefix)

`time::format` and `time::parse` accept `@name` shortcuts for Go's `time` package constants:

| Name | Example output |
|------|----------------|
| `@rfc3339` | `2006-01-02T15:04:05Z07:00` |
| `@rfc3339nano` | `2006-01-02T15:04:05.999999999Z07:00` |
| `@date` | `2006-01-02` |
| `@time` | `15:04:05` |
| `@datetime` | `2006-01-02 15:04:05` |
| `@rfc1123` | `Mon, 02 Jan 2006 15:04:05 MST` |
| `@rfc822` | `02 Jan 06 15:04 MST` |
| `@ansic`, `@unixdate`, `@rubydate`, `@rfc822z`, `@rfc850`, `@rfc1123z`, `@kitchen`, `@stamp`, `@stampmilli`, `@stampmicro`, `@stampnano` | (see Go `time` package) |

## Functions

### Timestamp — Creation

| Function | Signature | Description |
|----------|-----------|-------------|
| `time::now()` | `() → time` | Current time in local timezone |
| `time::now(tz)` | `(string) → time` | Current time in named IANA timezone |
| `time::parse(s)` | `(string) → time` | Parse RFC 3339 string |
| `time::parse(format, s)` | `(string, string) → time` | Parse with Go reference-time format or `@name` alias |
| `time::parse(format, s, tz)` | `(string, string, string) → time` | Parse with format; apply IANA timezone |
| `time::from_unix(n)` | `(number) → time` | Create time from Unix seconds (integer or fractional) in UTC |
| `time::from_unix(n, unit)` | `(number, string) → time` | Unit: `"s"`, `"ms"`, `"us"`, or `"ns"` |
| `time::strptime(format, s)` | `(string, string) → time` | Parse with strftime-style format |
| `time::strptime(format, s, tz)` | `(string, string, string) → time` | Parse with strftime format; apply IANA timezone |

### Timestamp — Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `time::format(format, t)` | `(string, time) → string` | Format with Go reference-time format or `@name` alias |
| `time::strftime(format, t)` | `(string, time) → string` | Format with strftime / C-style format |

### Timestamp — Arithmetic

| Function | Signature | Description |
|----------|-----------|-------------|
| `time::add(t, d)` | `(time, duration) → time` | Add a duration to a time. Either argument may be a string: a timestamp as RFC 3339, a duration in Go or ISO 8601 syntax. Always returns a `time`. |
| `time::sub(t1, t2)` | `(time, time) → duration` | Elapsed from `t2` to `t1`; negative if `t1 < t2` |
| `time::sub(t, d)` | `(time, duration) → time` | Subtract duration from time |
| `time::since(t)` | `(time) → duration` | Elapsed since `t` |
| `time::until(t)` | `(time) → duration` | Time remaining until `t` |
| `time::add_years(t, n)` | `(time, number) → time` | Add `n` calendar years |
| `time::add_months(t, n)` | `(time, number) → time` | Add `n` calendar months |
| `time::add_days(t, n)` | `(time, number) → time` | Add `n` calendar days |

### Timestamp — Decomposition

| Function | Signature | Description |
|----------|-----------|-------------|
| `time::to_unix(t)` | `(time) → number` | Unix epoch as fractional seconds |
| `time::to_unix(t, unit)` | `(time, string) → number` | Unix epoch in unit: `"s"` (float), `"ms"`, `"us"`, `"ns"` (integers) |
| `time::zone()` | `() → string` | System local timezone name |
| `time::zone(t)` | `(time) → string` | Stored timezone name |
| `time::in_zone(t, tz)` | `(time, string) → time` | Re-express `t` in given IANA timezone |

Calendar fields (`year`, `month`, `day`, `hour`, `minute`, `second`, `nanosecond`, `weekday`, `yearday`, `isoweek`, `isoyear`) are extracted via the rich-cty-types generic `get(t, part)` function — see [rich-cty-types integration](#rich-cty-types-integration).

### Timestamp — Comparison

go-cty v1.18 does not support ordering operators for capsule types. Use these functions instead:

| Function | Signature | Description |
|----------|-----------|-------------|
| `time::before(t1, t2)` | `(time, time) → bool` | True if `t1` is before `t2` |
| `time::after(t1, t2)` | `(time, time) → bool` | True if `t1` is after `t2` |

### Duration — Creation

| Function | Signature | Description |
|----------|-----------|-------------|
| `duration(s)` | `(string) → duration` | Parse ISO 8601 (`PT5M`) or Go format (`5m30s`) |
| `duration(n, unit)` | `(number, string) → duration` | `n` in given unit: `"h"`, `"m"`, `"s"`, `"ms"`, `"us"`, `"ns"` |

### Duration — Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `duration::format(d)` | `(duration) → string` | Go format (e.g. `"1h30m5s"`) |
| `duration::format(d, fmt)` | `(duration, string) → string` | `fmt` is `"go"` (default) or `"iso"` (ISO 8601 P-notation) |

### Duration — Arithmetic

Duration in a given unit is extracted via the rich-cty-types generic `get(d, unit)` function — see [rich-cty-types integration](#rich-cty-types-integration).

| Function | Signature | Description |
|----------|-----------|-------------|
| `duration::abs(d)` | `(duration) → duration` | Absolute value |
| `duration::add(d1, d2)` | `(duration, duration) → duration` | Sum |
| `duration::sub(d1, d2)` | `(duration, duration) → duration` | Difference |
| `duration::mul(d, n)` | `(duration, number) → duration` | Scale by factor |
| `duration::div(d, n)` | `(duration, number) → duration` | Divide by factor |
| `duration::truncate(d, m)` | `(duration, duration) → duration` | Truncate to multiple of `m` |
| `duration::round(d, m)` | `(duration, duration) → duration` | Round to nearest multiple of `m` |
| `duration::lt(d1, d2)` | `(duration, duration) → bool` | True if `d1 < d2` |
| `duration::gt(d1, d2)` | `(duration, duration) → bool` | True if `d1 > d2` |

### DNS Zone Serials

Functions for working with DNS zone serial numbers in `YYYYMMDDNN` format.

| Function | Signature | Description |
|----------|-----------|-------------|
| `dns::next_zone_serial(s)` | `(number\|string) → number` | Next serial after `s`, using today's date |
| `dns::next_zone_serial(s, t)` | `(number\|string, time) → number` | Next serial using date from `t` |
| `dns::parse_zone_serial(s)` | `(number\|string) → time` | Parse serial back to approximate date (UTC midnight) |

## Examples

```hcl
# Current time
time::now("UTC")
time::now("America/New_York")

# Parse
time::parse("2024-01-15T10:30:00Z")
time::parse("2006-01-02", "2024-01-15", "UTC")
time::strptime("%Y-%m-%d", "2024-01-15")

# Format
time::format("@date", time::now("UTC"))           # "2024-01-15"
time::format("2006-01-02", time::now("UTC"))      # same
time::strftime("%Y-%m-%d", time::now("UTC"))          # same

# Arithmetic
time::add(time::now("UTC"), duration("1h30m"))
time::sub(end_time, start_time)             # → duration
time::sub(deadline, duration(30, "m"))      # → time

# Duration
time::since(start_time)
get(time::since(start_time), "s")               # float seconds (requires rich-cty-types)
duration::format(time::since(start_time))         # "5m32s"
duration::format(time::since(start_time), "iso")  # "PT5M32S"
tostring(time::since(start_time))               # "5m32s" (requires rich-cty-types)

# Comparison
duration::gt(time::since(last_seen), duration(24, "h"))
time::before(expires_at, time::now("UTC"))

# Calendar field extraction (requires rich-cty-types)
get(time::now("UTC"), "year")                   # 2024
get(time::now("UTC"), "weekday")                # 0=Sun ... 6=Sat

# Unix interop
time::from_unix(epoch_seconds)
time::from_unix(epoch_ms, "ms")
time::to_unix(time::now("UTC"), "ns")

# Calendar
time::add_months(time::now("UTC"), 3)
time::add_days(time::now("UTC"), -7)

# DNS zone serials
dns::next_zone_serial(2026012300)                # → 2026012301
dns::next_zone_serial(old_serial, time::now("UTC"))    # → next serial for today
dns::parse_zone_serial(2026012307)               # → 2026-01-23 00:00:00 UTC
```

## License

BSD 2-Clause — see [LICENSE](LICENSE).
