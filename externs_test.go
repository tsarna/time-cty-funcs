package timecty

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// The extern files are scanned here with regexes rather than parsed with functy on
// purpose: this package must not depend on functy (its bytes are opaque to it), and
// the check only needs the names.
var (
	namespaceRE  = regexp.MustCompile(`(?m)^namespace (\w+)\s*$`)
	externDeclRE = regexp.MustCompile(`(?m)^func (\w+)\(`)
)

// declaredExterns returns every declared name, qualified by its file's namespace, and
// how many forms each has.
func declaredExterns(t *testing.T) map[string]int {
	t.Helper()
	declared := make(map[string]int)

	for file, src := range Externs() {
		var ns string
		if m := namespaceRE.FindStringSubmatch(string(src)); m != nil {
			ns = m[1] + "::"
		}
		for _, m := range externDeclRE.FindAllStringSubmatch(string(src), -1) {
			declared[ns+m[1]]++
		}
		require.True(t, strings.HasPrefix(string(src), "//functy:extern\n"),
			"%s must begin with the //functy:extern directive: functy's RegisterExterns "+
				"verifies it rather than forcing the mode, so that the file is also a valid "+
				"standalone .cty that `functy fmt` and `functy symbols` can open", file)
	}
	return declared
}

// A function needs an extern when its cty signature cannot tell the truth about it:
// it fakes an optional or defaulted argument with a variadic (cty can only make
// *trailing* parameters optional), or its arguments are a union or genuinely
// overloaded — shapes that a single cty signature has no way to say.
//
// The value is the number of forms declared, so that losing one is caught too.
var wantExterns = map[string]int{
	"time::now":              1,
	"time::parse":            3, // arg 0 is a timestamp at arity 1, a format at arity 2-3
	"time::strptime":         1,
	"time::from_unix":        1,
	"time::to_unix":          1,
	"time::zone":             1,
	"time::add":              4, // ts is time|string, dur is duration|string
	"time::sub":              2, // the return type flips between duration and time
	"duration":               2, // duration(s) vs duration(n, unit)
	"duration::format":       1,
	"dns::next_zone_serial":  2, // the serial may be a number or a string
	"dns::parse_zone_serial": 2,
}

// TestExternsCoverEveryMisrepresentedFunction is the drift guard. A function whose cty
// signature lies, and that has no extern, reflects as that lie — silently. A
// declaration for a function that no longer exists documents something nobody can
// call. Either way, this fails.
func TestExternsCoverEveryMisrepresentedFunction(t *testing.T) {
	declared := declaredExterns(t)
	funcs := GetTimeFunctions()

	for name, forms := range wantExterns {
		require.Contains(t, funcs, name, "wantExterns names %s(), which is not provided", name)
		assert.Equal(t, forms, declared[name],
			"%s() should have %d form(s) declared, found %d", name, forms, declared[name])
	}

	for name := range declared {
		assert.Contains(t, wantExterns, name,
			"the externs declare %s(), which is not in wantExterns", name)
		assert.Contains(t, funcs, name,
			"the externs declare %s(), which GetTimeFunctions does not provide", name)
	}
}

// One file per namespace, because a functy source declares at most one — so a name's
// forms cannot be split across files, and neither can a namespace.
func TestOneNamespacePerExternFile(t *testing.T) {
	seen := make(map[string]string)
	for file, src := range Externs() {
		m := namespaceRE.FindAllStringSubmatch(string(src), -1)
		require.LessOrEqual(t, len(m), 1, "%s declares more than one namespace", file)

		ns := "" // the global namespace, for the bare duration() constructor
		if len(m) == 1 {
			ns = m[0][1]
		}
		if prev, dup := seen[ns]; dup {
			t.Errorf("namespace %q is declared in both %s and %s; functy treats one name "+
				"across two files as a collision, not an overload set", ns, prev, file)
		}
		seen[ns] = file
	}
}

// Every function, and every parameter of every function, must carry a cty description
// — extern or not.
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
				"declare it in the externs and add it to wantExterns", name)

		for _, p := range fn.Params() {
			assert.NotEqual(t, cty.DynamicPseudoType, p.Type,
				"%s() parameter %q is DynamicPseudoType, which says nothing about what it accepts. "+
					"Either give it a concrete type, or — if it is genuinely a union — declare the "+
					"function in the externs as one form per type", name, p.Name)
		}
	}
}

// The names are namespaced, and `timeadd` is not among them: that name belongs to
// cty's own stdlib.TimeAddFunc, which a host registers directly. Shedding that
// compatibility duty is what lets every form of time::add return a time.
func TestNamesAreNamespaced(t *testing.T) {
	funcs := GetTimeFunctions()

	assert.NotContains(t, funcs, "timeadd",
		"timeadd belongs to cty's stdlib; this package provides time::add")

	for name := range funcs {
		if name == "duration" {
			continue // the type constructor, deliberately un-namespaced
		}
		assert.Contains(t, name, "::", "%s() is not namespaced", name)
	}
}

// The excess-argument ceiling. A cty VarParam has no upper bound of its own, so a
// function faking an optional argument with one used to accept unbounded trailing junk
// and drop it in silence: time::now("UTC", "junk") returned a time.
func TestExcessArgumentsAreRejected(t *testing.T) {
	junk := cty.StringVal("junk")
	now, err := NowFunc.Call(nil)
	require.NoError(t, err)

	for name, call := range map[string]func() error{
		"time::now": func() error {
			_, err := NowFunc.Call([]cty.Value{cty.StringVal("UTC"), junk})
			return err
		},
		"time::from_unix": func() error {
			_, err := FromUnixFunc.Call([]cty.Value{cty.NumberIntVal(0), cty.StringVal("s"), junk})
			return err
		},
		"time::to_unix": func() error {
			_, err := UnixFunc.Call([]cty.Value{now, cty.StringVal("s"), junk})
			return err
		},
		"duration::format": func() error {
			_, err := FormatDurationFunc.Call([]cty.Value{NewDurationCapsule(0), cty.StringVal("go"), junk})
			return err
		},
		"time::zone": func() error {
			_, err := TimezoneFunc.Call([]cty.Value{now, now})
			return err
		},
	} {
		assert.Error(t, call(), "%s() silently accepted a surplus argument", name)
	}
}
