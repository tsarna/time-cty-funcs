package timecty

import (
	"fmt"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// DurationFunc creates a duration from a string or from a number and unit.
// Called as duration("5m"), duration("PT5M"), or duration(5, "m").
var DurationFunc = function.New(&function.Spec{
	Description: `A duration, from a string ("1h30m" or ISO 8601 "PT1H30M") or from a number and a unit. The first argument's type therefore decides how many arguments there are, which is why the externs declare this as two forms.`,
	Params: []function.Parameter{
		{
			// A union (string | number), which cty cannot say; the Type func decides.
			Name:        "val",
			Type:        cty.DynamicPseudoType,
			Description: "A duration string, or the number of `unit`s.",
		},
	},
	// Present only in the (number, unit) form, so cty forces a variadic; but it is a
	// string, and now says so.
	VarParam: &function.Parameter{
		Name:        "unit",
		Type:        cty.String,
		Description: `One of "h", "m", "s", "ms", "us", "ns". Required when val is a number; not allowed when it is a string.`,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		switch len(args) {
		case 1:
			t := args[0].Type()
			if t != cty.String && t != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("duration() 1-arg form requires a string, got %s", t.FriendlyName())
			}
			return DurationCapsuleType, nil
		case 2:
			t0, t1 := args[0].Type(), args[1].Type()
			if t0 != cty.Number && t0 != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("duration() 2-arg form requires a number as first argument, got %s", t0.FriendlyName())
			}
			if t1 != cty.String && t1 != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("duration() 2-arg form requires a string unit as second argument, got %s", t1.FriendlyName())
			}
			return DurationCapsuleType, nil
		default:
			return cty.NilType, fmt.Errorf("duration() requires 1 or 2 arguments, got %d", len(args))
		}
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		if len(args) == 1 {
			return parseDurationString(args[0].AsString())
		}
		// 2-arg form: (number, unit)
		n, _ := args[0].AsBigFloat().Float64()
		return durationFromNumber(n, args[1].AsString())
	},
})

// FormatDurationFunc formats a duration as a string.
// Called as duration::format(d) for Go format (default) or duration::format(d, "iso") for ISO 8601.
var FormatDurationFunc = function.New(&function.Spec{
	Description: "Render a duration.",
	Params: []function.Parameter{
		{
			Name:        "d",
			Type:        DurationCapsuleType,
			Description: "The duration to render.",
		},
	},
	// Optional, defaulting to "go" — which cty cannot say. See the externs.
	VarParam: &function.Parameter{
		Name:        "fmt",
		Type:        cty.String,
		Description: `"go" ("1h30m0s") or "iso" ("PT1H30M"); defaults to "go".`,
	},
	Type: boundedArity("duration::format", 2, cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, err := GetDuration(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		format := "go"
		if len(args) > 1 {
			format = args[1].AsString()
		}
		switch format {
		case "go", "":
			return cty.StringVal(d.String()), nil
		case "iso":
			return cty.StringVal(durationToISO8601(d)), nil
		default:
			return cty.NilVal, fmt.Errorf("duration::format: unknown format %q; valid values are \"go\" and \"iso\"", format)
		}
	},
})

// AbsDurationFunc returns the absolute value of a duration.
var AbsDurationFunc = function.New(&function.Spec{
	Description: "A duration's magnitude, discarding its sign.",
	Params: []function.Parameter{
		{
			Name:        "d",
			Type:        DurationCapsuleType,
			Description: "The duration.",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, err := GetDuration(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		if d < 0 {
			d = -d
		}
		return NewDurationCapsule(d), nil
	},
})

// --- Duration arithmetic ---

// DurationAddFunc adds two durations: d1 + d2
var DurationAddFunc = function.New(&function.Spec{
	Description: "The sum of two durations.",
	Params: []function.Parameter{
		{
			Name:        "d1",
			Type:        DurationCapsuleType,
			Description: "The first duration.",
		},
		{
			Name:        "d2",
			Type:        DurationCapsuleType,
			Description: "The duration to add.",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d1, _ := GetDuration(args[0])
		d2, _ := GetDuration(args[1])
		return NewDurationCapsule(d1 + d2), nil
	},
})

// DurationSubFunc subtracts durations: d1 - d2
var DurationSubFunc = function.New(&function.Spec{
	Description: "The difference between two durations. Negative if d2 is the longer.",
	Params: []function.Parameter{
		{
			Name:        "d1",
			Type:        DurationCapsuleType,
			Description: "The duration to subtract from.",
		},
		{
			Name:        "d2",
			Type:        DurationCapsuleType,
			Description: "The duration to subtract.",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d1, _ := GetDuration(args[0])
		d2, _ := GetDuration(args[1])
		return NewDurationCapsule(d1 - d2), nil
	},
})

// DurationMulFunc multiplies a duration by a scalar: d * n
var DurationMulFunc = function.New(&function.Spec{
	Description: "A duration scaled by a factor.",
	Params: []function.Parameter{
		{
			Name:        "d",
			Type:        DurationCapsuleType,
			Description: "The duration to scale.",
		},
		{
			Name:        "n",
			Type:        cty.Number,
			Description: "The factor; may be fractional.",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, _ := GetDuration(args[0])
		n, _ := args[1].AsBigFloat().Float64()
		return NewDurationCapsule(time.Duration(float64(d) * n)), nil
	},
})

// DurationDivFunc divides a duration by a scalar: d / n (returns duration)
var DurationDivFunc = function.New(&function.Spec{
	Description: "A duration divided by a divisor. Dividing by zero is an error.",
	Params: []function.Parameter{
		{
			Name:        "d",
			Type:        DurationCapsuleType,
			Description: "The duration to divide.",
		},
		{
			Name:        "n",
			Type:        cty.Number,
			Description: "The divisor; may be fractional.",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, _ := GetDuration(args[0])
		n, _ := args[1].AsBigFloat().Float64()
		if n == 0 {
			return cty.NilVal, fmt.Errorf("duration::div: division by zero")
		}
		return NewDurationCapsule(time.Duration(float64(d) / n)), nil
	},
})

// DurationTruncateFunc truncates d to a multiple of m: d.Truncate(m)
var DurationTruncateFunc = function.New(&function.Spec{
	Description: "A duration rounded down to a multiple of m.",
	Params: []function.Parameter{
		{
			Name:        "d",
			Type:        DurationCapsuleType,
			Description: "The duration to truncate.",
		},
		{
			Name:        "m",
			Type:        DurationCapsuleType,
			Description: "The multiple to truncate to — itself a duration, e.g. duration(\"1m\").",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, _ := GetDuration(args[0])
		m, _ := GetDuration(args[1])
		return NewDurationCapsule(d.Truncate(m)), nil
	},
})

// DurationRoundFunc rounds d to the nearest multiple of m: d.Round(m)
var DurationRoundFunc = function.New(&function.Spec{
	Description: "A duration rounded to the nearest multiple of m, halves away from zero.",
	Params: []function.Parameter{
		{
			Name:        "d",
			Type:        DurationCapsuleType,
			Description: "The duration to round.",
		},
		{
			Name:        "m",
			Type:        DurationCapsuleType,
			Description: "The multiple to round to — itself a duration, e.g. duration(\"1m\").",
		},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, _ := GetDuration(args[0])
		m, _ := GetDuration(args[1])
		return NewDurationCapsule(d.Round(m)), nil
	},
})

// --- Duration comparison ---

// DurationLtFunc returns true if d1 < d2.
var DurationLtFunc = function.New(&function.Spec{
	Description: "Whether d1 is shorter than d2. Signed: -1h is less than 1s.",
	Params: []function.Parameter{
		{
			Name:        "d1",
			Type:        DurationCapsuleType,
			Description: "The duration to test.",
		},
		{
			Name:        "d2",
			Type:        DurationCapsuleType,
			Description: "The duration to compare against.",
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d1, err := GetDuration(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		d2, err := GetDuration(args[1])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.BoolVal(d1 < d2), nil
	},
})

// DurationGtFunc returns true if d1 > d2.
var DurationGtFunc = function.New(&function.Spec{
	Description: "Whether d1 is longer than d2. Signed: 1s is greater than -1h.",
	Params: []function.Parameter{
		{
			Name:        "d1",
			Type:        DurationCapsuleType,
			Description: "The duration to test.",
		},
		{
			Name:        "d2",
			Type:        DurationCapsuleType,
			Description: "The duration to compare against.",
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d1, err := GetDuration(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		d2, err := GetDuration(args[1])
		if err != nil {
			return cty.NilVal, err
		}
		return cty.BoolVal(d1 > d2), nil
	},
})
