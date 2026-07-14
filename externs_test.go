package timecty

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// externDeclRE matches a top-level declaration in externs.cty. The file is scanned
// here with a regex rather than parsed with functy on purpose: this package must not
// depend on functy (its bytes are opaque to it), and the check only needs the names.
var externDeclRE = regexp.MustCompile(`(?m)^func (\w+)\(`)

func declaredExterns() map[string]int {
	declared := make(map[string]int)
	for _, m := range externDeclRE.FindAllStringSubmatch(string(Externs()), -1) {
		declared[m[1]]++
	}
	return declared
}

// A function needs an extern when its cty signature cannot tell the truth about it:
// it fakes an optional or defaulted argument with a variadic (cty can only make
// *trailing* parameters optional), or it is genuinely overloaded — its argument
// shapes, or its return type, differ per call.
//
// The value is the number of forms declared, so that losing one is caught too.
var wantExterns = map[string]int{
	"now":             1,
	"parsetime":       3, // arg 0 is a timestamp at arity 1, a format at arity 2-3
	"strptime":        1,
	"fromunix":        1,
	"unix":            1,
	"timezone":        1,
	"duration":        2, // duration(s) vs duration(n, unit)
	"formatduration":  1,
	"timeadd":         4, // return type flips between string and time
	"timesub":         2, // return type flips between duration and time
	"nextzoneserial":  2, // the serial may be a number or a string
	"parsezoneserial": 2,
}

// TestExternsCoverEveryMisrepresentedFunction is the drift guard. A function whose
// cty signature lies, and that has no extern, reflects as that lie — silently. A
// declaration for a function that no longer exists documents something nobody can
// call. Either way, this fails.
func TestExternsCoverEveryMisrepresentedFunction(t *testing.T) {
	declared := declaredExterns()
	funcs := GetTimeFunctions()

	for name, forms := range wantExterns {
		require.Contains(t, funcs, name, "wantExterns names %s(), which is not provided", name)
		assert.Equal(t, forms, declared[name],
			"%s() should have %d form(s) declared in externs.cty, found %d", name, forms, declared[name])
	}

	for name := range declared {
		assert.Contains(t, wantExterns, name,
			"externs.cty declares %s(), which is not in wantExterns", name)
		assert.Contains(t, funcs, name,
			"externs.cty declares %s(), which GetTimeFunctions does not provide", name)
	}
}

// The bytes must declare themselves an extern file: functy's RegisterExterns verifies
// the directive rather than forcing the mode, so that this same file is a valid
// standalone .cty that `functy fmt` and `functy symbols` can open.
func TestExternsCarryTheDirective(t *testing.T) {
	require.True(t, strings.HasPrefix(string(Externs()), "//functy:extern\n"),
		"externs.cty must begin with the //functy:extern directive")
}

// Every function, and every parameter of every function, must carry a cty
// description — extern or not.
//
// The cty metadata is the only documentation a non-functy cty host can see, and the
// only thing functy's own doc() reads (doc() does not consult the extern), so a gap
// here reads as "exists but undocumented" even where help() shows a full block. An
// extern says what a signature *is*; it does not excuse the metadata from saying what
// the function does.
func TestEverythingIsDescribed(t *testing.T) {
	for name, fn := range GetTimeFunctions() {
		assert.NotEmpty(t, fn.Description(), "%s() has no cty Description", name)

		for _, p := range fn.Params() {
			assert.NotEmpty(t, p.Description, "%s() parameter %q has no Description", name, p.Name)
		}
		if vp := fn.VarParam(); vp != nil {
			assert.NotEmpty(t, vp.Description, "%s() variadic parameter %q has no Description", name, vp.Name)
		}
	}
}

// A function with no extern is one whose cty signature is the whole truth. So it must
// stay one: no variadic (which would mean it had started faking an optional argument),
// and no dynamic parameter (which says nothing about what it accepts). Either would
// mean the function now needs an extern, and this is what says so.
func TestFunctionsWithoutExternsHaveHonestSignatures(t *testing.T) {
	for name, fn := range GetTimeFunctions() {
		if _, hasExtern := wantExterns[name]; hasExtern {
			continue
		}

		assert.Nil(t, fn.VarParam(),
			"%s() has grown a VarParam, so its cty signature can no longer be honest; "+
				"declare it in externs.cty and add it to wantExterns", name)

		for _, p := range fn.Params() {
			assert.NotEqual(t, cty.DynamicPseudoType, p.Type,
				"%s() parameter %q is DynamicPseudoType, which says nothing about what it accepts. "+
					"Either give it a concrete type, or — if it is genuinely a union — declare the "+
					"function in externs.cty as one form per type", name, p.Name)
		}
	}
}

// The excess-argument ceiling. A cty VarParam has no upper bound of its own, so a
// function faking an optional argument with one used to accept unbounded trailing
// junk and drop it in silence: now("UTC", "junk") returned a time.
func TestExcessArgumentsAreRejected(t *testing.T) {
	tz := cty.StringVal("UTC")
	junk := cty.StringVal("junk")
	now, err := NowFunc.Call(nil)
	require.NoError(t, err)

	for name, call := range map[string]func() error{
		"now": func() error { _, err := NowFunc.Call([]cty.Value{tz, junk}); return err },
		"fromunix": func() error {
			_, err := FromUnixFunc.Call([]cty.Value{cty.NumberIntVal(0), cty.StringVal("s"), junk})
			return err
		},
		"unix": func() error { _, err := UnixFunc.Call([]cty.Value{now, cty.StringVal("s"), junk}); return err },
		"formatduration": func() error {
			_, err := FormatDurationFunc.Call([]cty.Value{NewDurationCapsule(0), cty.StringVal("go"), junk})
			return err
		},
		"timezone": func() error { _, err := TimezoneFunc.Call([]cty.Value{now, now}); return err },
	} {
		assert.Error(t, call(), "%s() silently accepted a surplus argument", name)
	}
}
