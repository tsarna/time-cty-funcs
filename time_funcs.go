package timecty

import (
	"fmt"
	"time"

	timefmt "github.com/itchyny/timefmt-go"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// NowFunc returns the current time, optionally in the given IANA timezone.
// Called as time::now() or time::now("America/New_York").
//
// The optional tz is a VarParam because cty has no other way to spell an optional
// argument — see the externs, which declare the signature this cannot.
var NowFunc = function.New(&function.Spec{
	Description: "The current time. Without a zone, in the machine's local zone.",
	VarParam: &function.Parameter{
		Name:        "tz",
		Type:        cty.String,
		Description: `IANA zone name, e.g. "America/New_York".`,
	},
	Type: boundedArity("time::now", 1, TimeCapsuleType),
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
//	time::parse(s)              — RFC 3339 (timezone required)
//	time::parse(format, s)      — parse s using Go layout (or @name alias)
//	time::parse(format, s, tz)  — same, but interpret s in the given IANA timezone
var ParseTimeFunc = function.New(&function.Spec{
	Description: "Parse a timestamp. With one argument it reads RFC 3339; with two, the first is a Go reference layout or an @alias; a third interprets the parsed wall-clock in that zone. The meaning of the first argument therefore depends on how many there are, which is why the externs declare this as three forms.",
	// Every argument is in the variadic, because there is no other way to say that
	// arg 0 means something different depending on how many there are.
	VarParam: &function.Parameter{
		Name:        "args",
		Type:        cty.String,
		Description: "The timestamp; or a format and then the timestamp; or a format, the timestamp, and an IANA zone.",
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) < 1 || len(args) > 3 {
			return cty.NilType, fmt.Errorf("time::parse() takes 1 to 3 arguments")
		}
		return TimeCapsuleType, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		switch len(args) {
		case 1:
			s := args[0].AsString()
			t, err := time.Parse(time.RFC3339Nano, s)
			if err != nil {
				return cty.NilVal, fmt.Errorf("time::parse: invalid RFC 3339 timestamp %q: %s", s, err)
			}
			return NewTimeCapsule(t), nil
		case 2:
			layout, err := resolveFormat(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			t, err := time.Parse(layout, args[1].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("time::parse: cannot parse %q with format %q: %s", args[1].AsString(), args[0].AsString(), err)
			}
			return NewTimeCapsule(t), nil
		case 3:
			layout, err := resolveFormat(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			loc, err := time.LoadLocation(args[2].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("time::parse: invalid timezone %q: %s", args[2].AsString(), err)
			}
			t, err := time.ParseInLocation(layout, args[1].AsString(), loc)
			if err != nil {
				return cty.NilVal, fmt.Errorf("time::parse: cannot parse %q with format %q: %s", args[1].AsString(), args[0].AsString(), err)
			}
			return NewTimeCapsule(t), nil
		default:
			return cty.NilVal, fmt.Errorf("time::parse() takes 1 to 3 arguments")
		}
	},
})

// TimeAddFunc adds a duration to a time, and always returns a time.
//
// Signatures:
//
//	time::add(time, duration)   → time
//	time::add(time, string)     → time   (string parsed as a duration)
//	time::add(string, duration) → time   (string parsed as RFC 3339 Nano)
//	time::add(string, string)   → time
//
// It carries no stdlib-compatibility burden: cty's own stdlib.TimeAddFunc is the
// string-in/string-out `timeadd`, and a host that wants that behavior registers it
// directly. Shedding that duty is what lets every form here return a time — so the
// return type is static, and a string is parsed the same way whichever form it lands
// in. It used to be neither: (string, string) returned a *string* and accepted only
// Go duration syntax, so time::add("2024-01-01T00:00:00Z", "PT5M") failed while
// time::add(time::parse("2024-01-01T00:00:00Z"), "PT5M") succeeded.
var TimeAddFunc = function.New(&function.Spec{
	Description: "Add a duration to a time. Either argument may be given as a string: a timestamp as RFC 3339, a duration in Go syntax (\"1h30m\") or ISO 8601 (\"PT1H30M\").",
	Params: []function.Parameter{
		{
			// Genuinely a union (time | string), which cty has no way to say; the Type
			// func decides. The extern declares one form per combination.
			Name:        "ts",
			Type:        cty.DynamicPseudoType,
			Description: "The time to add to: a time, or an RFC 3339 string.",
		},
		{
			Name:        "dur",
			Type:        cty.DynamicPseudoType,
			Description: "The duration to add: a duration, or a string in Go or ISO 8601 syntax.",
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		t0, t1 := args[0].Type(), args[1].Type()
		// Unknown types at check time — defer to runtime
		if t0 == cty.DynamicPseudoType || t1 == cty.DynamicPseudoType {
			return cty.DynamicPseudoType, nil
		}
		validTS := t0 == TimeCapsuleType || t0 == cty.String
		validDur := t1 == DurationCapsuleType || t1 == cty.String
		if validTS && validDur {
			return TimeCapsuleType, nil
		}
		return cty.NilType, fmt.Errorf("time::add: unsupported argument types %s and %s", t0.FriendlyName(), t1.FriendlyName())
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		// The time: a capsule, or an RFC 3339 (Nano) string.
		var t time.Time
		switch args[0].Type() {
		case cty.String:
			var err error
			t, err = time.Parse(time.RFC3339Nano, args[0].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("time::add: invalid timestamp %q: %s", args[0].AsString(), err)
			}
		case TimeCapsuleType:
			var err error
			t, err = GetTime(args[0])
			if err != nil {
				return cty.NilVal, err
			}
		default:
			return cty.NilVal, fmt.Errorf("time::add: first argument must be a time or string, got %s", args[0].Type().FriendlyName())
		}

		// The duration: a capsule, or a string in Go or ISO 8601 syntax.
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
			return cty.NilVal, fmt.Errorf("time::add: second argument must be a duration or string, got %s", args[1].Type().FriendlyName())
		}

		return NewTimeCapsule(t.Add(d)), nil
	},
})

// TimeSubFunc subtracts a time or duration from a time.
//
// Signatures:
//
//	time::sub(time, time)     → duration   (elapsed from t2 to t1; negative if t1 < t2)
//	time::sub(time, duration) → time       (time minus duration)
var TimeSubFunc = function.New(&function.Spec{
	Description: "Subtract from a time: another time, giving the duration between them (negative if t2 is later), or a duration, giving the earlier time. The return type therefore depends on the arguments, which is why the externs declare this as two forms.",
	Params: []function.Parameter{
		{
			Name:        "t1",
			Type:        TimeCapsuleType,
			Description: "The time to subtract from.",
		},
		{
			// Genuinely a union (time | duration) and therefore not sayable in cty,
			// which has no union type — the Type func below decides. the externs
			// declares the two forms.
			Name:        "t2",
			Type:        cty.DynamicPseudoType,
			Description: "A time (giving the duration between) or a duration (giving the earlier time).",
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		t1 := args[1].Type()
		if t1 == cty.DynamicPseudoType {
			return cty.DynamicPseudoType, nil
		}
		switch t1 {
		case TimeCapsuleType:
			return DurationCapsuleType, nil
		case DurationCapsuleType:
			return TimeCapsuleType, nil
		default:
			return cty.NilType, fmt.Errorf("time::sub: second argument must be a time or duration, got %s", t1.FriendlyName())
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
			return cty.NilVal, fmt.Errorf("time::sub: second argument must be a time or duration, got %s", args[1].Type().FriendlyName())
		}
	},
})

// SinceFunc returns the duration elapsed since the given time (equivalent to time::sub(time::now(), t)).
var SinceFunc = function.New(&function.Spec{
	Description: "How long ago a time was: the duration from it until now. Negative if it is in the future.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to measure from.",
		},
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

// UntilFunc returns the duration until the given time (equivalent to time::sub(t, time::now())).
var UntilFunc = function.New(&function.Spec{
	Description: "How far off a time is: the duration from now until it. Negative if it has passed.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to measure to.",
		},
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
// Called as time::format("2006-01-02", t) or time::format("@rfc3339", t).
var FormatTimeFunc = function.New(&function.Spec{
	Description: "Render a time with a Go reference layout.",
	Params: []function.Parameter{
		{
			Name:        "format",
			Type:        cty.String,
			Description: "A Go reference layout (\"2006-01-02\"), or an @alias such as @rfc3339.",
		},
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to render.",
		},
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
// Called as time::strftime("%Y-%m-%d", t).
var StrftimeFunc = function.New(&function.Spec{
	Description: "Render a time with a C-style strftime format. Unlike formattime, @aliases are not resolved here.",
	Params: []function.Parameter{
		{
			Name:        "format",
			Type:        cty.String,
			Description: "A strftime-style layout, e.g. \"%Y-%m-%d\".",
		},
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to render.",
		},
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
// Called as time::strptime("%Y-%m-%d", "2024-01-15") or time::strptime("%Y-%m-%d", "2024-01-15", "UTC").
var StrptimeFunc = function.New(&function.Spec{
	Description: "Parse a timestamp with a C-style strftime format. Unlike parsetime and formattime, @aliases are not resolved here.",
	Params: []function.Parameter{
		{
			Name:        "format",
			Type:        cty.String,
			Description: "strftime-style layout, e.g. \"%Y-%m-%d\".",
		},
		{
			Name:        "s",
			Type:        cty.String,
			Description: "The timestamp to parse.",
		},
	},
	// Optional — hence a variadic, which is all cty offers. See the externs.
	VarParam: &function.Parameter{
		Name:        "tz",
		Type:        cty.String,
		Description: "IANA zone; the parsed wall-clock is read as being in it. Optional.",
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 3 {
			return cty.NilType, fmt.Errorf("time::strptime() takes 2 or 3 arguments")
		}
		return TimeCapsuleType, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := timefmt.Parse(args[1].AsString(), args[0].AsString())
		if err != nil {
			return cty.NilVal, fmt.Errorf("time::strptime: cannot parse %q with format %q: %s", args[1].AsString(), args[0].AsString(), err)
		}
		if len(args) == 3 {
			loc, err := time.LoadLocation(args[2].AsString())
			if err != nil {
				return cty.NilVal, fmt.Errorf("time::strptime: invalid timezone %q: %s", args[2].AsString(), err)
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
// Called as time::from_unix(n) for seconds (possibly fractional), or time::from_unix(n, unit)
// where unit is "s", "ms", "us", or "ns". Always returns UTC.
var FromUnixFunc = function.New(&function.Spec{
	Description: "A time from a Unix epoch count. Always returns a UTC time.",
	Params: []function.Parameter{
		{
			Name:        "n",
			Type:        cty.Number,
			Description: "The epoch count; fractional seconds are honored.",
		},
	},
	// The unit is optional and defaults to "s" — which cty cannot say, so it is faked
	// with a variadic. the externs declares the real signature.
	VarParam: &function.Parameter{
		Name:        "unit",
		Type:        cty.String,
		Description: `One of "s", "ms", "us", "ns"; defaults to "s".`,
	},
	Type: boundedArity("time::from_unix", 2, TimeCapsuleType),
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
			return cty.NilVal, fmt.Errorf("time::from_unix: unknown unit %q; valid units: s, ms, us, ns", unit)
		}
	},
})

// UnixFunc returns the Unix epoch value for a time.
// Called as time::to_unix(t) for fractional seconds, or time::to_unix(t, unit) where unit is
// "s" (float), "ms", "us", or "ns" (integers).
var UnixFunc = function.New(&function.Spec{
	Description: "A time as a Unix epoch count. Seconds are fractional; the other units are whole.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to convert.",
		},
	},
	// Optional, defaulting to "s"; see FromUnixFunc.
	VarParam: &function.Parameter{
		Name:        "unit",
		Type:        cty.String,
		Description: `One of "s", "ms", "us", "ns"; defaults to "s".`,
	},
	Type: boundedArity("time::to_unix", 2, cty.Number),
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
			return cty.NilVal, fmt.Errorf("time::to_unix: unknown unit %q; valid units: s, ms, us, ns", unit)
		}
	},
})

// --- Timezone ---

// TimezoneFunc returns the timezone name.
// Called as time::zone() for the local system timezone, or time::zone(t) for the
// timezone stored in a time value.
var TimezoneFunc = function.New(&function.Spec{
	Description: "A time's zone name. Without an argument, the machine's local zone.",
	// Optional, so cty forces a variadic; but it is a time, and now says so — cty
	// itself rejects anything else, where a hand-rolled check in the Type func used to.
	VarParam: &function.Parameter{
		Name:        "t",
		Type:        TimeCapsuleType,
		Description: "The time whose zone to report; omit for the local zone.",
	},
	Type: boundedArity("time::zone", 1, cty.String),
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
	Description: "The same instant, displayed in another zone.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to convert.",
		},
		{
			Name:        "tz",
			Type:        cty.String,
			Description: "IANA zone name, e.g. \"America/New_York\".",
		},
	},
	Type: function.StaticReturnType(TimeCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		t, err := GetTime(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		loc, err := time.LoadLocation(args[1].AsString())
		if err != nil {
			return cty.NilVal, fmt.Errorf("time::in_zone: invalid timezone %q: %s", args[1].AsString(), err)
		}
		return NewTimeCapsule(t.In(loc)), nil
	},
})

// --- Calendar arithmetic ---

// AddYearsFunc adds n calendar years to a time (calls time.Time.AddDate).
var AddYearsFunc = function.New(&function.Spec{
	Description: "A time shifted by whole calendar years. Negative n moves backwards.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to shift.",
		},
		{
			Name:        "n",
			Type:        cty.Number,
			Description: "Years to add; truncated to a whole number.",
		},
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
	Description: "A time shifted by whole calendar months. Negative n moves backwards; a day-of-month that the target month lacks rolls forward, as Go's AddDate does.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to shift.",
		},
		{
			Name:        "n",
			Type:        cty.Number,
			Description: "Months to add; truncated to a whole number.",
		},
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
	Description: "A time shifted by whole days. Negative n moves backwards.",
	Params: []function.Parameter{
		{
			Name:        "t",
			Type:        TimeCapsuleType,
			Description: "The time to shift.",
		},
		{
			Name:        "n",
			Type:        cty.Number,
			Description: "Days to add; truncated to a whole number.",
		},
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
	Description: "Whether t1 is earlier than t2.",
	Params: []function.Parameter{
		{
			Name:        "t1",
			Type:        TimeCapsuleType,
			Description: "The time to test.",
		},
		{
			Name:        "t2",
			Type:        TimeCapsuleType,
			Description: "The time to compare against.",
		},
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
	Description: "Whether t1 is later than t2.",
	Params: []function.Parameter{
		{
			Name:        "t1",
			Type:        TimeCapsuleType,
			Description: "The time to test.",
		},
		{
			Name:        "t2",
			Type:        TimeCapsuleType,
			Description: "The time to compare against.",
		},
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
