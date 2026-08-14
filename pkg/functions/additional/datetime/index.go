// Package datetime provides strftime-style time formatting, sparing users from Go's reference-time layout ("2006-01-02
// 15:04:05")
package datetime

import (
	"fmt"
	"strings"
	"text/template"
	"time"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"strftime": strftime,
	}
}

// maps strftime tokens to Go's reference-time layout tokens. Tokens that don't have a direct Go layout equivalent (%s,
// %e, %%) are handled separately in specialToken
var tokenLayouts = map[byte]string{
	'Y': "2006",
	'y': "06",
	'm': "01",
	'd': "02",
	'H': "15",
	'I': "03",
	'M': "04",
	'S': "05",
	'p': "PM",
	'A': "Monday",
	'a': "Mon",
	'B': "January",
	'b': "Jan",
	'j': "002",
	'Z': "MST",
	'z': "-0700",
}

// formats a time value using strftime-style tokens, sparing users from Go's reference-time layout ("2006-01-02
// 15:04:05")
//
// The time argument accepts any of:
//   - time.Time / *time.Time
//   - int / int64 (Unix epoch seconds)
//   - string (RFC3339 or "2006-01-02")
//
// For "now" use sprig's `now` in a pipeline: {{ now | strftime "%Y-%m-%d" }}
//
// Supported tokens (subset of C strftime):
//
//	%Y  4-digit year          %y  2-digit year
//	%m  month (01-12)         %B  full month name    %b  abbrev month
//	%d  day of month (01-31)  %e  day of month space-padded
//	%j  day of year (001-366)
//	%H  hour 24 (00-23)       %I  hour 12 (01-12)    %p  AM/PM
//	%M  minute (00-59)        %S  second (00-59)
//	%A  full weekday          %a  abbrev weekday
//	%Z  timezone name         %z  timezone offset (-0700)
//	%s  Unix epoch seconds
//	%%  literal %
//
// Unknown %-tokens are passed through verbatim. No locale-aware weekday /
// month names (English only) and no sub-second tokens (%N, %f)
func strftime(format string, t any) (string, error) {
	tm, err := toTime(t)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(len(format))
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			b.WriteByte(format[i])
			continue
		}
		if i+1 >= len(format) {
			b.WriteByte('%')
			break
		}
		c := format[i+1]
		i++
		if layout, ok := tokenLayouts[c]; ok {
			b.WriteString(tm.Format(layout))
			continue
		}
		writeSpecialToken(&b, tm, c)
	}
	return b.String(), nil
}

// handles strftime tokens that don't map to a single Go layout token: %s (epoch), %e (space-padded day), %% (literal
// percent), and unknown tokens which are passed through verbatim
func writeSpecialToken(b *strings.Builder, tm time.Time, c byte) {
	switch c {
	case 's':
		fmt.Fprintf(b, "%d", tm.Unix())
	case 'e':
		fmt.Fprintf(b, "%2d", tm.Day())
	case '%':
		b.WriteByte('%')
	default:
		b.WriteByte('%')
		b.WriteByte(c)
	}
}

// coerces various time representations into a time.Time
func toTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case *time.Time:
		return *t, nil
	case int64:
		return time.Unix(t, 0), nil
	case int:
		return time.Unix(int64(t), 0), nil
	case string:
		if tm, err := time.Parse(time.RFC3339, t); err == nil {
			return tm, nil
		}
		if tm, err := time.Parse("2006-01-02", t); err == nil {
			return tm, nil
		}
		return time.Time{}, fmt.Errorf("strftime: cannot parse time string %q", t)
	default:
		return time.Time{}, fmt.Errorf("strftime: unsupported time type %T", v)
	}
}
