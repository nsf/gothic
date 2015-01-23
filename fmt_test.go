package gothic

import (
	"bytes"
	"regexp"
	"testing"
)

func test_format(t *testing.T, gold, format string, args ...interface{}) {
	var buf bytes.Buffer
	err := sprintf(&buf, format, args...)
	if err != nil {
		t.Error(err)
		return
	}
	s := buf.String()
	if gold != s {
		t.Errorf("%q != %q", gold, s)
	}
}

func must_contain(t *testing.T, err error, re string) {
	if err == nil {
		t.Error("non-nil error expected")
		return
	}
	r := regexp.MustCompile(re)
	if !r.MatchString(err.Error()) {
		t.Errorf("%q doesn't contain %q", err, re)
	}
}

func test_error(t *testing.T, gold, format string, args ...interface{}) {
	var buf bytes.Buffer
	err := sprintf(&buf, format, args...)
	must_contain(t, err, gold)
}

func TestFormat(t *testing.T) {
	am := ArgMap{"i": 10, "j": 5}
	test_format(t, "simple as is %{oops}", "simple as is %{oops}")
	test_format(t, "10 = 5 + 5", "%{} = %{} + %{1}", 10, 5)
	test_format(t, "10 5 5", "%{i} %{j} %{j}", am)
	test_format(t, "3.14", "%{%.2f}", 3.1415)
	test_format(t, "005", "%{j%03d}", am)
	test_format(t, `"\[command \$variable\]"`, "%{%q}", "[command $variable]")
	test_error(t, "missing enclosing bracket", "%{} %{", 10, 5)
	test_error(t, "not-a-number", "%{oops}", 10, 5)
	test_error(t, "there is no.+index -100", "%{-100}", 1, 2, 3)
	test_error(t, "there is no.+index 100", "%{100}", 1, 2, 3)
	test_error(t, "empty format tag", "%{}", am)
	test_error(t, `no argument "x "`, "%{i} %{x }", am)
}

func test_quote(t *testing.T, gold, s string) {
	var buf bytes.Buffer
	quote(&buf, s)
	s2 := buf.String()
	if gold != s2 {
		t.Errorf("%s != %s (%q)", gold, s2, s)
	}
}

func TestQuote(t *testing.T) {
	test_quote(t, `"\[command \$variable\]"`, "[command $variable]")
	test_quote(t, `"\{1 2 3\}"`, "{1 2 3}")
	test_quote(t, `"\a\b\f\n\r\t\v\x00"`, "\a\b\f\n\r\t\v\x00")
}
