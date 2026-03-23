package timecty

import (
	"fmt"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// NextZoneSerialFunc computes the next DNS zone serial number in YYYYMMDDNN format.
//
// Called as nextzoneserial(s) or nextzoneserial(s, t).
//
//	s: current serial (number or string)
//	t: optional time capsule; defaults to now()
//
// Computes x = first serial of the day for t (YYYYMMDD * 100), then returns max(s+1, x).
var NextZoneSerialFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "s", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "t",
		Type: cty.DynamicPseudoType,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 2 {
			return cty.NilType, fmt.Errorf("nextzoneserial() takes 1 or 2 arguments")
		}
		t0 := args[0].Type()
		if t0 != cty.Number && t0 != cty.String && t0 != cty.DynamicPseudoType {
			return cty.NilType, fmt.Errorf("nextzoneserial: serial must be a number or string, got %s", t0.FriendlyName())
		}
		if len(args) == 2 {
			t1 := args[1].Type()
			if t1 != TimeCapsuleType && t1 != cty.DynamicPseudoType {
				return cty.NilType, fmt.Errorf("nextzoneserial: second argument must be a time value, got %s", t1.FriendlyName())
			}
		}
		return cty.Number, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		s, err := parseSerialArg(args[0], "nextzoneserial")
		if err != nil {
			return cty.NilVal, err
		}
		var t time.Time
		if len(args) == 2 {
			t, err = GetTime(args[1])
			if err != nil {
				return cty.NilVal, err
			}
		} else {
			t = time.Now()
		}
		year, month, day := t.Date()
		x := int64(year)*1_000_000 + int64(month)*10_000 + int64(day)*100
		return cty.NumberIntVal(max(s+1, x)), nil
	},
})

// ParseZoneSerialFunc converts a DNS zone serial back to an approximate time value.
// The serial format is YYYYMMDDNN; the NN sequence number is ignored.
// For out-of-range date components, the nearest valid date is used:
// month > 12 → December 31; day > days in month → last day of month.
var ParseZoneSerialFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "s", Type: cty.DynamicPseudoType},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		t := args[0].Type()
		if t != cty.Number && t != cty.String && t != cty.DynamicPseudoType {
			return cty.NilType, fmt.Errorf("parsezoneserial: serial must be a number or string, got %s", t.FriendlyName())
		}
		return TimeCapsuleType, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		s, err := parseSerialArg(args[0], "parsezoneserial")
		if err != nil {
			return cty.NilVal, err
		}
		datepart := s / 100
		year := int(datepart / 10_000)
		month := time.Month((datepart / 100) % 100)
		day := int(datepart % 100)

		// Snap invalid components to the nearest valid date.
		if month < 1 {
			month = 1
		}
		if month > 12 {
			month = 12
			day = 31 // last day of December — already valid
		}
		if day < 1 {
			day = 1
		}
		if last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day(); day > last {
			day = last
		}
		return NewTimeCapsule(time.Date(year, month, day, 0, 0, 0, 0, time.UTC)), nil
	},
})
