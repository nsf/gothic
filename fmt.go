package gothic

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A special type which can be passed to Interpreter.Eval method family as the
// only argument and in that case you can use named abbreviations within format
// tags.
type ArgMap map[string]interface{}

func split_tag(tag string) (abbrev, format string) {
	abbrev = tag
	format = ""
	if i := strings.Index(tag, "%"); i != -1 {
		abbrev = tag[:i]
		format = tag[i:]
	}
	return
}

func write_arg_quoted(buf *bytes.Buffer, arg interface{}) {
	switch a := arg.(type) {
	case string:
		quote(buf, a)
	case error:
		quote(buf, a.Error())
	case fmt.Stringer:
		quote(buf, a.String())
	default:
		// TODO: it doesn't work in all cases, we still need to escape
		// various $ { } [ ] symbols
		fmt.Fprintf(buf, "%q", arg)
	}
}

func write_arg(buf *bytes.Buffer, arg interface{}, format string) {
	if format != "" {
		if format == "%q" {
			write_arg_quoted(buf, arg)
			return
		} else {
			fmt.Fprintf(buf, format, arg)
		}
	} else {
		fmt.Fprint(buf, arg)
	}
}

func write_tag(buf *bytes.Buffer, tag string, counter *int, args []interface{}) error {
	argnum := 0
	if tag == "" || strings.HasPrefix(tag, "%") {
		// no abbrev, means use counter
	}

	abbrev, format := split_tag(tag)
	if abbrev == "" {
		// no abbrev, means use the counter
		argnum = *counter
		(*counter)++
	} else {
		// non-empty abbrev, let's convert it to the integer
		i, err := strconv.ParseInt(abbrev, 10, 16)
		if err != nil {
			return errors.New("gothic.sprintf: not-a-number tag abbrev")
		}

		argnum = int(i)
	}

	if argnum < 0 || argnum >= len(args) {
		return fmt.Errorf("gothic.sprintf: there is no argument with index %d", argnum)
	}

	write_arg(buf, args[argnum], format)
	return nil
}

func write_tag_argmap(buf *bytes.Buffer, tag string, argmap ArgMap) error {
	if tag == "" || strings.HasPrefix(tag, "%") {
		return errors.New("gothic.sprintf: empty format tag abbrev on gothic.ArgMap call form")
	}

	key, format := split_tag(tag)
	arg, found := argmap[key]
	if !found {
		return fmt.Errorf("gothic.sprintf: no argument %q in the ArgMap", key)
	}

	write_arg(buf, arg, format)
	return nil
}

func sprintf(buf *bytes.Buffer, format string, args ...interface{}) error {
	if len(args) == 0 {
		// quick path
		buf.WriteString(format)
		return nil
	}

	counter := 0
	argmap := false
	argmapvar := ArgMap(nil)
	if len(args) == 1 {
		argmapvar, argmap = args[0].(ArgMap)
	}

	offset := 0
	for {
		i := strings.Index(format[offset:], "%{")
		if i == -1 {
			// no more format tags, write the rest and return
			buf.WriteString(format[offset:])
			break
		}

		// write everything before the formatter
		buf.WriteString(format[offset : offset+i])

		// now let's fine the ending "}"
		b := offset + i + 2
		j := strings.Index(format[b:], "}")
		if j == -1 {
			return errors.New("gothic.sprintf: missing enclosing bracket in a formatter tag")
		}
		e := b + j

		var err error
		tag := format[b:e]
		if argmap {
			err = write_tag_argmap(buf, tag, argmapvar)
		} else {
			err = write_tag(buf, tag, &counter, args)
		}
		if err != nil {
			return err
		}
		offset = e + 1
	}
	return nil
}

func quote_rune(buf *bytes.Buffer, r rune, size int) {
	const lowerhex = "0123456789abcdef"
	if size == 1 && r == utf8.RuneError {
		// invalid rune, write the byte as is
		buf.WriteString(`\x`)
		buf.WriteByte(lowerhex[r>>4])
		buf.WriteByte(lowerhex[r&0xF])
		return
	}

	// first check for special TCL escaping cases
	switch r {
	case '{', '}', '[', ']', '"', '$', '\\':
		buf.WriteString("\\")
		buf.WriteRune(r)
		return
	}

	// other printable characters
	if unicode.IsPrint(r) {
		buf.WriteRune(r)
		return
	}

	// non-printable characters
	switch r {
	case '\a':
		buf.WriteString(`\a`)
	case '\b':
		buf.WriteString(`\b`)
	case '\f':
		buf.WriteString(`\f`)
	case '\n':
		buf.WriteString(`\n`)
	case '\r':
		buf.WriteString(`\r`)
	case '\t':
		buf.WriteString(`\t`)
	case '\v':
		buf.WriteString(`\v`)
	default:
		switch {
		case r < ' ':
			buf.WriteString(`\x`)
			buf.WriteByte(lowerhex[r>>4])
			buf.WriteByte(lowerhex[r&0xF])
		case r >= 0x10000:
			r = 0xFFFD
			fallthrough
		case r < 0x10000:
			buf.WriteString(`\u`)
			for s := 12; s >= 0; s -= 4 {
				buf.WriteByte(lowerhex[r>>uint(s)&0xF])
			}
		}
	}
}

func quote(buf *bytes.Buffer, s string) {
	buf.WriteString(`"`)
	size := 0
	for offset := 0; offset < len(s); offset += size {
		r := rune(s[offset])
		size = 1
		if r >= utf8.RuneSelf {
			r, size = utf8.DecodeRuneInString(s[offset:])
		}

		quote_rune(buf, r, size)
	}
	buf.WriteString(`"`)
}

// Works exactly like Eval("%{%q}"), but instead of evaluating returns a quoted
// string.
func Quote(s string) string {
	var tmp bytes.Buffer
	quote(&tmp, s)
	return tmp.String()
}

// Quotes the rune just like if it was passed through Quote, the result is the
// same as: Quote(string(r)).
func QuoteRune(r rune) string {
	var tmp bytes.Buffer
	size := utf8.RuneLen(r)
	tmp.WriteString(`"`)
	quote_rune(&tmp, r, size)
	tmp.WriteString(`"`)
	return tmp.String()
}
