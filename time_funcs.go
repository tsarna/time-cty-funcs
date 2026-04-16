package timecty

import (
	"fmt"
	"time"

	timefmt "github.com/itchyny/timefmt-go"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// NowFunc returns the current time, optionally in the given IANA timezone.
// Called as now() or now("America/New_York").
var NowFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{
		Name: "tz",
		Type: cty.String,
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		if len(args) == 0 {
			return NewTimeCapsule(time.Now()), nil
		}
		tzName := args[0].AsString()
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			return cty.NilVal, fmt.Errorf("invalid timezone %q: %s", tzName, err)
		}
		return NewTimeCapsule(time.Now().In(loc)), nil
	},
})

// ParseTimeFunc parses a timestamp string into a time value.
//
// Forms:
//
//	parsetime(s)              — RFC 3339 (timezone required)
//	parsetime(format, s)      — parse s using Go layout (or @name alias)
//	parsetime(format, s, tz)  — same, but interpret s in the given IANA timezone
var ParseTimeFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{Name: "args", Type: cty.String},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) < 1 || len(args) > 3 {
			return cty.NilType, fmt.Errorf("parsetime() takes 1 to 3 arguments")
		}
		return TimeCapsuleType, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		switch len(args) {
		case 1:
			s := args[0].AsString()
			t, err := time.Parse(time.RFC3339Nano, s)
			if err != nil {
				return cty.NilVal, fmt.Errorf("parsetime: invalid RFC 3339 timestamp %q: %s", s, err)
			}
			return NewTimeCapsule(t), nil
		case 2:
			layout, err := resolveFormat(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			t, err := time.Parse(layout, args[1].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("parsetime: cannot parse %q with format %q: %s", args[1].AsString(), args[0].AsString(), err)
			}
			return NewTimeCapsule(t), nil
		case 3:
			layout, err := resolveFormat(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			loc, err := time.LoadLocation(args[2].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("parsetime: invalid timezone %q: %s", args[2].AsString(), err)
			}
			t, err := time.ParseInLocation(layout, args[1].AsString(), loc)
			if err != nil {
				return cty.NilVal, fmt.Errorf("parsetime: cannot parse %q with format %q: %s", args[1].AsString(), args[0].AsString(), err)
			}
			return NewTimeCapsule(t), nil
		default:
			return cty.NilVal, fmt.Errorf("parsetime() takes 1 to 3 arguments")
		}
	},
})

// TimeAddFunc adds a duration to a time. Backward-compatible with the stdlib
// timeadd(string, string) form; also accepts capsule types.
//
// Signatures:
//
//	timeadd(string, string) → string   (standard hcl behavior)
//	timeadd(time, duration) → time
//	timeadd(time, string)   → time     (string auto-parsed as duration)
//	timeadd(string, duration) → time   (string auto-parsed as RFC 3339)
var TimeAddFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "ts", Type: cty.DynamicPseudoType},
		{Name: "dur", Type: cty.DynamicPseudoType},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		t0, t1 := args[0].Type(), args[1].Type()
		// Unknown types at check time — defer to runtime
		if t0 == cty.DynamicPseudoType || t1 == cty.DynamicPseudoType {
			return cty.DynamicPseudoType, nil
		}
		// (string, string) — backward-compatible path returns string
		if t0 == cty.String && t1 == cty.String {
			return cty.String, nil
		}
		// All other valid combinations return time
		validTS := t0 == TimeCapsuleType || t0 == cty.String
		validDur := t1 == DurationCapsuleType || t1 == cty.String
		if validTS && validDur {
			return TimeCapsuleType, nil
		}
		return cty.NilType, fmt.Errorf("timeadd: unsupported argument types %s and %s", t0.FriendlyName(), t1.FriendlyName())
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		// (string, string) — backward-compatible behavior preserved exactly
		if args[0].Type() == cty.String && args[1].Type() == cty.String {
			ts, err := time.Parse(time.RFC3339, args[0].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("timeadd: invalid timestamp %q: %s", args[0].AsString(), err)
			}
			dur, err := time.ParseDuration(args[1].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("timeadd: invalid duration %q: %s", args[1].AsString(), err)
			}
			return cty.StringVal(ts.Add(dur).Format(time.RFC3339)), nil
		}

		// Get the time value
		var t time.Time
		switch args[0].Type() {
		case cty.String:
			var err error
			t, err = time.Parse(time.RFC3339Nano, args[0].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("timeadd: invalid timestamp %q: %s", args[0].AsString(), err)
			}
		case TimeCapsuleType:
			var err error
			t, err = GetTime(args[0])
			if err != nil {
				return cty.NilVal, err
			}
		default:
			return cty.NilVal, fmt.Errorf("timeadd: first argument must be a time or string, got %s", args[0].Type().FriendlyName())
		}

		// Get the duration value
		var d time.Duration
		switch args[1].Type() {
		case cty.String:
			v, err := parseDurationString(args[1].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			d, err = GetDuration(v)
			if err != nil {
				return cty.NilVal, err
			}
		case DurationCapsuleType:
			var err error
			d, err = GetDuration(args[1])
			if err != nil {
				return cty.NilVal, err
			}
		default:
			return cty.NilVal, fmt.Errorf("timeadd: second argument must be a duration or string, got %s", args[1].Type().FriendlyName())
		}

		return NewTimeCapsule(t.Add(d)), nil
	},
})

// TimeSubFunc subtracts a time or duration from a time.
//
// Signatures:
//
//	timesub(time, time)     → duration   (elapsed from t2 to t1; negative if t1 < t2)
//	timesub(time, duration) → time       (time minus duration)
var TimeSubFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t1", Type: cty.DynamicPseudoType},
		{Name: "t2", Type: cty.DynamicPseudoType},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		t0, t1 := args[0].Type(), args[1].Type()
		if t0 == cty.DynamicPseudoType || t1 == cty.DynamicPseudoType {
			return cty.DynamicPseudoType, nil
		}
		if t0 != TimeCapsuleType {
			return cty.NilType, fmt.Errorf("timesub: first argument must be a time, got %s", t0.FriendlyName())
		}
		switch t1 {
		case TimeCapsuleType:
			return DurationCapsuleType, nil
		case DurationCapsuleType:
			return TimeCapsuleType, nil
		default:
			return cty.NilType, fmt.Errorf("timesub: second argument must be a time or duration, got %s", t1.FriendlyName())
		}
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t1, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		switch args[1].Type() {
		case TimeCapsuleType:
			t2, err := GetTime(args[1])
			if err != nil {
				return cty.NilVal, err
			}
			return NewDurationCapsule(t1.Sub(t2)), nil
		case DurationCapsuleType:
			d, err := GetDuration(args[1])
			if err != nil {
				return cty.NilVal, err
			}
			return NewTimeCapsule(t1.Add(-d)), nil
		default:
			return cty.NilVal, fmt.Errorf("timesub: second argument must be a time or duration, got %s", args[1].Type().FriendlyName())
		}
	},
})

// SinceFunc returns the duration elapsed since the given time (equivalent to timesub(now(), t)).
var SinceFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		return NewDurationCapsule(time.Since(t)), nil
	},
})

// UntilFunc returns the duration until the given time (equivalent to timesub(t, now())).
var UntilFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		return NewDurationCapsule(time.Until(t)), nil
	},
})

// FormatTimeFunc formats a time value using Go's reference-time format or a @name alias.
// Called as formattime("2006-01-02", t) or formattime("@rfc3339", t).
var FormatTimeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "format", Type: cty.String},
		{Name: "t", Type: TimeCapsuleType},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		layout, err := resolveFormat(args[0].AsString())
		if err != nil {
			return cty.NilVal, err
		}
		t, err := GetTime(args[1])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(t.Format(layout)), nil
	},
})

// StrftimeFunc formats a time using a strftime-style format string (via itchyny/timefmt-go).
// Called as strftime("%Y-%m-%d", t).
var StrftimeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "format", Type: cty.String},
		{Name: "t", Type: TimeCapsuleType},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[1])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(timefmt.Format(t, args[0].AsString())), nil
	},
})

// StrptimeFunc parses a time string using a strftime-style format (via itchyny/timefmt-go).
// Called as strptime("%Y-%m-%d", "2024-01-15") or strptime("%Y-%m-%d", "2024-01-15", "UTC").
var StrptimeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "format", Type: cty.String},
		{Name: "s", Type: cty.String},
	},
	VarParam: &function.Parameter{Name: "tz", Type: cty.String},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 3 {
			return cty.NilType, fmt.Errorf("strptime() takes 2 or 3 arguments")
		}
		return TimeCapsuleType, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := timefmt.Parse(args[1].AsString(), args[0].AsString())
		if err != nil {
			return cty.NilVal, fmt.Errorf("strptime: cannot parse %q with format %q: %s", args[1].AsString(), args[0].AsString(), err)
		}
		if len(args) == 3 {
			loc, err := time.LoadLocation(args[2].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("strptime: invalid timezone %q: %s", args[2].AsString(), err)
			}
			// Reinterpret the parsed wall-clock components as being in the given timezone,
			// rather than converting the UTC instant.
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
		}
		return NewTimeCapsule(t), nil
	},
})

// --- Unix interop ---

// FromUnixFunc creates a time from a Unix epoch value.
// Called as fromunix(n) for seconds (possibly fractional), or fromunix(n, unit)
// where unit is "s", "ms", "us", or "ns". Always returns UTC.
var FromUnixFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "n", Type: cty.Number},
	},
	VarParam: &function.Parameter{
		Name: "unit",
		Type: cty.String,
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		unit := "s"
		if len(args) > 1 {
			unit = args[1].AsString()
		}
		n, _ := args[0].AsBigFloat().Float64()
		switch unit {
		case "s":
			secs := int64(n)
			nanos := int64((n - float64(secs)) * 1e9)
			return NewTimeCapsule(time.Unix(secs, nanos).UTC()), nil
		case "ms":
			return NewTimeCapsule(time.UnixMilli(int64(n)).UTC()), nil
		case "us":
			return NewTimeCapsule(time.UnixMicro(int64(n)).UTC()), nil
		case "ns":
			return NewTimeCapsule(time.Unix(0, int64(n)).UTC()), nil
		default:
			return cty.NilVal, fmt.Errorf("fromunix: unknown unit %q; valid units: s, ms, us, ns", unit)
		}
	},
})

// UnixFunc returns the Unix epoch value for a time.
// Called as unix(t) for fractional seconds, or unix(t, unit) where unit is
// "s" (float), "ms", "us", or "ns" (integers).
var UnixFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
	},
	VarParam: &function.Parameter{
		Name: "unit",
		Type: cty.String,
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		unit := "s"
		if len(args) > 1 {
			unit = args[1].AsString()
		}
		switch unit {
		case "s":
			return cty.NumberFloatVal(float64(t.UnixNano()) / 1e9), nil
		case "ms":
			return cty.NumberIntVal(t.UnixMilli()), nil
		case "us":
			return cty.NumberIntVal(t.UnixMicro()), nil
		case "ns":
			return cty.NumberIntVal(t.UnixNano()), nil
		default:
			return cty.NilVal, fmt.Errorf("unix: unknown unit %q; valid units: s, ms, us, ns", unit)
		}
	},
})

// --- Timezone ---

// TimezoneFunc returns the timezone name.
// Called as timezone() for the local system timezone, or timezone(t) for the
// timezone stored in a time value.
var TimezoneFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{
		Name: "t",
		Type: cty.DynamicPseudoType,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 1 {
			return cty.NilType, fmt.Errorf("timezone() takes 0 or 1 arguments")
		}
		if len(args) == 1 {
			t := args[0].Type()
			if t != TimeCapsuleType && t != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("timezone: argument must be a time value, got %s", t.FriendlyName())
			}
		}
		return cty.String, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		if len(args) == 0 {
			return cty.StringVal(time.Local.String()), nil
		}
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(t.Location().String()), nil
	},
})

// InTimezoneFunc re-expresses a time in a different IANA timezone.
// The instant is unchanged; only the displayed timezone changes.
var InTimezoneFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
		{Name: "tz", Type: cty.String},
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		loc, err := time.LoadLocation(args[1].AsString())
		if err != nil {
			return cty.NilVal, fmt.Errorf("intimezone: invalid timezone %q: %s", args[1].AsString(), err)
		}
		return NewTimeCapsule(t.In(loc)), nil
	},
})

// --- Calendar arithmetic ---

// AddYearsFunc adds n calendar years to a time (calls time.Time.AddDate).
var AddYearsFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
		{Name: "n", Type: cty.Number},
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		n, _ := args[1].AsBigFloat().Int64()
		return NewTimeCapsule(t.AddDate(int(n), 0, 0)), nil
	},
})

// AddMonthsFunc adds n calendar months to a time (calls time.Time.AddDate).
var AddMonthsFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
		{Name: "n", Type: cty.Number},
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		n, _ := args[1].AsBigFloat().Int64()
		return NewTimeCapsule(t.AddDate(0, int(n), 0)), nil
	},
})

// AddDaysFunc adds n calendar days to a time (calls time.Time.AddDate).
var AddDaysFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t", Type: TimeCapsuleType},
		{Name: "n", Type: cty.Number},
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		n, _ := args[1].AsBigFloat().Int64()
		return NewTimeCapsule(t.AddDate(0, 0, int(n))), nil
	},
})

// --- Comparison functions ---
// go-cty does not dispatch </>/<= etc. to capsule types, so ordering comparisons
// are provided as explicit functions.

// TimeBeforeFunc returns true if t1 is before t2.
var TimeBeforeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t1", Type: TimeCapsuleType},
		{Name: "t2", Type: TimeCapsuleType},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t1, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		t2, err := GetTime(args[1])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.BoolVal(t1.Before(t2)), nil
	},
})

// TimeAfterFunc returns true if t1 is after t2.
var TimeAfterFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "t1", Type: TimeCapsuleType},
		{Name: "t2", Type: TimeCapsuleType},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t1, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		t2, err := GetTime(args[1])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.BoolVal(t1.After(t2)), nil
	},
})
