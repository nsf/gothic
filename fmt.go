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

func write_arg(buf *bytes.Buffer, arg interface{}, format string) {
	if format != "" {
		if s, ok := arg.(string); ok && format == "%q" {
			// special case, quote string in a TCL specific way
			quote(buf, s)
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

func quote(buf *bytes.Buffer, s string) {
	const lowerhex = "0123456789abcdef"
	buf.WriteString(`"`)
	size := 0
	for offset := 0; offset < len(s); offset += size {
		r := rune(s[offset])
		size = 1
		if r >= utf8.RuneSelf {
			r, size = utf8.DecodeRuneInString(s[offset:])
		}

		if size == 1 && r == utf8.RuneError {
			// invalid rune, write the byte as is
			buf.WriteString(`\x`)
			buf.WriteByte(lowerhex[r>>4])
			buf.WriteByte(lowerhex[r&0xF])
			continue
		}

		// first check for special TCL escaping cases
		switch r {
		case '{', '}', '[', ']', '"', '$', '\\':
			buf.WriteString("\\")
			buf.WriteString(s[offset : offset+size])
			continue
		}

		// other printable characters
		if unicode.IsPrint(r) {
			buf.WriteString(s[offset : offset+size])
			continue
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
	buf.WriteString(`"`)
}
