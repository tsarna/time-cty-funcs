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
	Params: []function.Parameter{
		{Name: "val", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "unit",
		Type: cty.DynamicPseudoType,
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
// Called as formatduration(d) for Go format (default) or formatduration(d, "iso") for ISO 8601.
var FormatDurationFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "d", Type: DurationCapsuleType},
	},
	VarParam: &function.Parameter{
		Name: "fmt",
		Type: cty.String,
	},
	Type: function.StaticReturnType(cty.String),
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
			return cty.NilVal, fmt.Errorf("formatduration: unknown format %q; valid values are \"go\" and \"iso\"", format)
		}
	},
})

// AbsDurationFunc returns the absolute value of a duration.
var AbsDurationFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "d", Type: DurationCapsuleType},
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
	Params: []function.Parameter{
		{Name: "d1", Type: DurationCapsuleType},
		{Name: "d2", Type: DurationCapsuleType},
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
	Params: []function.Parameter{
		{Name: "d1", Type: DurationCapsuleType},
		{Name: "d2", Type: DurationCapsuleType},
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
	Params: []function.Parameter{
		{Name: "d", Type: DurationCapsuleType},
		{Name: "n", Type: cty.Number},
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
	Params: []function.Parameter{
		{Name: "d", Type: DurationCapsuleType},
		{Name: "n", Type: cty.Number},
	},
	Type: function.StaticReturnType(DurationCapsuleType),
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		d, _ := GetDuration(args[0])
		n, _ := args[1].AsBigFloat().Float64()
		if n == 0 {
			return cty.NilVal, fmt.Errorf("durationdiv: division by zero")
		}
		return NewDurationCapsule(time.Duration(float64(d) / n)), nil
	},
})

// DurationTruncateFunc truncates d to a multiple of m: d.Truncate(m)
var DurationTruncateFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "d", Type: DurationCapsuleType},
		{Name: "m", Type: DurationCapsuleType},
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
	Params: []function.Parameter{
		{Name: "d", Type: DurationCapsuleType},
		{Name: "m", Type: DurationCapsuleType},
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
	Params: []function.Parameter{
		{Name: "d1", Type: DurationCapsuleType},
		{Name: "d2", Type: DurationCapsuleType},
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
	Params: []function.Parameter{
		{Name: "d1", Type: DurationCapsuleType},
		{Name: "d2", Type: DurationCapsuleType},
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
