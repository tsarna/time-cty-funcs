# time-cty-funcs

cty functions and types for dealing with time; mainly used in HCL2 templates.

[![CI](https://github.com/tsarna/time-cty-funcs/actions/workflows/ci.yml/badge.svg)](https://github.com/tsarna/time-cty-funcs/actions/workflows/ci.yml)

## Overview

This package provides two [go-cty](https://github.com/zclconf/go-cty) capsule types — `time` and `duration` — plus a comprehensive set of functions for working with them in HCL2 expression evaluation contexts.

## Types

### `timecty.TimeCapsuleType`

A cty capsule type wrapping Go's `time.Time`. Supports equality (`==`, `!=`) via `CapsuleOps`. Timezone is stored inside the value; comparison is always by absolute UTC instant regardless of stored timezone.

### `timecty.DurationCapsuleType`

A cty capsule type wrapping Go's `time.Duration` (int64 nanoseconds; range ±~292 years). Supports equality (`==`, `!=`) via `CapsuleOps`. Use `durationlt`/`durationgt` (or extract via `get(d, unit)` and compare numerically) for ordering.

**Limitation:** Go's `time.Duration` cannot represent calendar months or years exactly. ISO 8601 durations like `P1Y` or `P1M` are rejected; use `addyears()` / `addmonths()` instead.

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

`GetTimeFunctions()` returns the functions described below. The `timeadd` entry supersedes the go-cty stdlib version, adding capsule-type support while remaining backward-compatible with the `(string, string) → string` form.

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

- `tostring(t)` formats a `time` as RFC 3339 with nanosecond precision (equivalent to `formattime("@rfc3339nano", t)`).
- `tostring(d)` formats a `duration` using Go syntax (equivalent to `formatduration(d)`).
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

`formattime` and `parsetime` accept `@name` shortcuts for Go's `time` package constants:

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
| `now()` | `() → time` | Current time in local timezone |
| `now(tz)` | `(string) → time` | Current time in named IANA timezone |
| `parsetime(s)` | `(string) → time` | Parse RFC 3339 string |
| `parsetime(format, s)` | `(string, string) → time` | Parse with Go reference-time format or `@name` alias |
| `parsetime(format, s, tz)` | `(string, string, string) → time` | Parse with format; apply IANA timezone |
| `fromunix(n)` | `(number) → time` | Create time from Unix seconds (integer or fractional) in UTC |
| `fromunix(n, unit)` | `(number, string) → time` | Unit: `"s"`, `"ms"`, `"us"`, or `"ns"` |
| `strptime(format, s)` | `(string, string) → time` | Parse with strftime-style format |
| `strptime(format, s, tz)` | `(string, string, string) → time` | Parse with strftime format; apply IANA timezone |

### Timestamp — Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `formattime(format, t)` | `(string, time) → string` | Format with Go reference-time format or `@name` alias |
| `strftime(format, t)` | `(string, time) → string` | Format with strftime/C-style format |

### Timestamp — Arithmetic

| Function | Signature | Description |
|----------|-----------|-------------|
| `timeadd(t, d)` | `(time, duration) → time` | Add duration to time (also accepts string forms for backward compat) |
| `timesub(t1, t2)` | `(time, time) → duration` | Elapsed from `t2` to `t1`; negative if `t1 < t2` |
| `timesub(t, d)` | `(time, duration) → time` | Subtract duration from time |
| `since(t)` | `(time) → duration` | Elapsed since `t` |
| `until(t)` | `(time) → duration` | Time remaining until `t` |
| `addyears(t, n)` | `(time, number) → time` | Add `n` calendar years |
| `addmonths(t, n)` | `(time, number) → time` | Add `n` calendar months |
| `adddays(t, n)` | `(time, number) → time` | Add `n` calendar days |

### Timestamp — Decomposition

| Function | Signature | Description |
|----------|-----------|-------------|
| `unix(t)` | `(time) → number` | Unix epoch as fractional seconds |
| `unix(t, unit)` | `(time, string) → number` | Unix epoch in unit: `"s"` (float), `"ms"`, `"us"`, `"ns"` (integers) |
| `timezone()` | `() → string` | System local timezone name |
| `timezone(t)` | `(time) → string` | Stored timezone name |
| `intimezone(t, tz)` | `(time, string) → time` | Re-express `t` in given IANA timezone |

Calendar fields (`year`, `month`, `day`, `hour`, `minute`, `second`, `nanosecond`, `weekday`, `yearday`, `isoweek`, `isoyear`) are extracted via the rich-cty-types generic `get(t, part)` function — see [rich-cty-types integration](#rich-cty-types-integration).

### Timestamp — Comparison

go-cty v1.18 does not support ordering operators for capsule types. Use these functions instead:

| Function | Signature | Description |
|----------|-----------|-------------|
| `timebefore(t1, t2)` | `(time, time) → bool` | True if `t1` is before `t2` |
| `timeafter(t1, t2)` | `(time, time) → bool` | True if `t1` is after `t2` |

### Duration — Creation

| Function | Signature | Description |
|----------|-----------|-------------|
| `duration(s)` | `(string) → duration` | Parse ISO 8601 (`PT5M`) or Go format (`5m30s`) |
| `duration(n, unit)` | `(number, string) → duration` | `n` in given unit: `"h"`, `"m"`, `"s"`, `"ms"`, `"us"`, `"ns"` |

### Duration — Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `formatduration(d)` | `(duration) → string` | Go format (e.g. `"1h30m5s"`) |
| `formatduration(d, fmt)` | `(duration, string) → string` | `fmt` is `"go"` (default) or `"iso"` (ISO 8601 P-notation) |

### Duration — Arithmetic

Duration in a given unit is extracted via the rich-cty-types generic `get(d, unit)` function — see [rich-cty-types integration](#rich-cty-types-integration).

| Function | Signature | Description |
|----------|-----------|-------------|
| `absduration(d)` | `(duration) → duration` | Absolute value |
| `durationadd(d1, d2)` | `(duration, duration) → duration` | Sum |
| `durationsub(d1, d2)` | `(duration, duration) → duration` | Difference |
| `durationmul(d, n)` | `(duration, number) → duration` | Scale by factor |
| `durationdiv(d, n)` | `(duration, number) → duration` | Divide by factor |
| `durationtruncate(d, m)` | `(duration, duration) → duration` | Truncate to multiple of `m` |
| `durationround(d, m)` | `(duration, duration) → duration` | Round to nearest multiple of `m` |
| `durationlt(d1, d2)` | `(duration, duration) → bool` | True if `d1 < d2` |
| `durationgt(d1, d2)` | `(duration, duration) → bool` | True if `d1 > d2` |

### DNS Zone Serials

Functions for working with DNS zone serial numbers in `YYYYMMDDNN` format.

| Function | Signature | Description |
|----------|-----------|-------------|
| `nextzoneserial(s)` | `(number\|string) → number` | Next serial after `s`, using today's date |
| `nextzoneserial(s, t)` | `(number\|string, time) → number` | Next serial using date from `t` |
| `parsezoneserial(s)` | `(number\|string) → time` | Parse serial back to approximate date (UTC midnight) |

## Examples

```hcl
# Current time
now("UTC")
now("America/New_York")

# Parse
parsetime("2024-01-15T10:30:00Z")
parsetime("2006-01-02", "2024-01-15", "UTC")
strptime("%Y-%m-%d", "2024-01-15")

# Format
formattime("@date", now("UTC"))           # "2024-01-15"
formattime("2006-01-02", now("UTC"))      # same
strftime("%Y-%m-%d", now("UTC"))          # same

# Arithmetic
timeadd(now("UTC"), duration("1h30m"))
timesub(end_time, start_time)             # → duration
timesub(deadline, duration(30, "m"))      # → time

# Duration
since(start_time)
get(since(start_time), "s")               # float seconds (requires rich-cty-types)
formatduration(since(start_time))         # "5m32s"
formatduration(since(start_time), "iso")  # "PT5M32S"
tostring(since(start_time))               # "5m32s" (requires rich-cty-types)

# Comparison
durationgt(since(last_seen), duration(24, "h"))
timebefore(expires_at, now("UTC"))

# Calendar field extraction (requires rich-cty-types)
get(now("UTC"), "year")                   # 2024
get(now("UTC"), "weekday")                # 0=Sun ... 6=Sat

# Unix interop
fromunix(epoch_seconds)
fromunix(epoch_ms, "ms")
unix(now("UTC"), "ns")

# Calendar
addmonths(now("UTC"), 3)
adddays(now("UTC"), -7)

# DNS zone serials
nextzoneserial(2026012300)                # → 2026012301
nextzoneserial(old_serial, now("UTC"))    # → next serial for today
parsezoneserial(2026012307)               # → 2026-01-23 00:00:00 UTC
```

## License

BSD 2-Clause — see [LICENSE](LICENSE).
