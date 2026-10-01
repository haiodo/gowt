package jrt

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// java.util.Calendar/Date and java.text.DateFormat (the part gtk's DateTime uses) over time. One fixed
// English/US locale: Locale arguments are ignored; add CLDR data here if another locale is needed.

const (
	CalendarERA          = 0
	CalendarYEAR         = 1
	CalendarMONTH        = 2
	CalendarDAY_OF_MONTH = 5
	CalendarDAY_OF_WEEK  = 7
	CalendarAM_PM        = 9
	CalendarHOUR         = 10
	CalendarHOUR_OF_DAY  = 11
	CalendarMINUTE       = 12
	CalendarSECOND       = 13
	CalendarMILLISECOND  = 14
	CalendarAM           = 0
	CalendarPM           = 1
	CalendarSHORT        = 1
	CalendarLONG         = 2

	DateFormatFULL   = 0
	DateFormatLONG   = 1
	DateFormatMEDIUM = 2
	DateFormatSHORT  = 3

	CharacterIteratorDONE uint16 = 0xFFFF
)

type Date struct{ t time.Time }

// Calendar normalizes raw fields on read, like a lenient GregorianCalendar (Jan 31 + month 1 = Mar 3).
type Calendar struct{ y, mo, d, h, mi, s, ms int32 }

func CalendarGetInstance(locale ...any) *Calendar {
	c := &Calendar{}
	c.setFrom(time.Now())
	return c
}

func (c *Calendar) setFrom(t time.Time) {
	c.y, c.mo, c.d = int32(t.Year()), int32(t.Month())-1, int32(t.Day())
	c.h, c.mi, c.s, c.ms = int32(t.Hour()), int32(t.Minute()), int32(t.Second()), int32(t.Nanosecond()/1e6)
}

func (c *Calendar) norm() time.Time {
	t := time.Date(int(c.y), time.Month(c.mo+1), int(c.d), int(c.h), int(c.mi), int(c.s), int(c.ms)*1e6, time.Local)
	c.setFrom(t)
	return t
}

func (c *Calendar) Clone() any { n := *c; return &n }
func (c *Calendar) GetTime() *Date   { return &Date{c.norm()} }
func (c *Calendar) SetTime(d *Date)  { c.setFrom(d.t) }

func (c *Calendar) Get(field int32) int32 {
	c.norm()
	switch field {
	case CalendarERA:
		return 1
	case CalendarYEAR:
		return c.y
	case CalendarMONTH:
		return c.mo
	case CalendarDAY_OF_MONTH:
		return c.d
	case CalendarDAY_OF_WEEK:
		return int32(c.norm().Weekday()) + 1
	case CalendarAM_PM:
		return c.h / 12
	case CalendarHOUR:
		return c.h % 12
	case CalendarHOUR_OF_DAY:
		return c.h
	case CalendarMINUTE:
		return c.mi
	case CalendarSECOND:
		return c.s
	case CalendarMILLISECOND:
		return c.ms
	}
	panic(&IllegalArgumentException{Message: "Calendar field " + strconv.Itoa(int(field))})
}

// Set is set(field, value) or set(year, month, day).
func (c *Calendar) Set(a ...int32) {
	if len(a) == 3 {
		c.y, c.mo, c.d = a[0], a[1], a[2]
		return
	}
	v := a[1]
	switch a[0] {
	case CalendarYEAR:
		c.y = v
	case CalendarMONTH:
		c.mo = v
	case CalendarDAY_OF_MONTH:
		c.d = v
	case CalendarAM_PM:
		c.norm()
		c.h = c.h%12 + 12*v
	case CalendarHOUR:
		c.norm()
		c.h = c.h/12*12 + v
	case CalendarHOUR_OF_DAY:
		c.h = v
	case CalendarMINUTE:
		c.mi = v
	case CalendarSECOND:
		c.s = v
	case CalendarMILLISECOND:
		c.ms = v
	default:
		panic(&IllegalArgumentException{Message: "Calendar field " + strconv.Itoa(int(a[0]))})
	}
}

func daysIn(y, mo int32) int32 { return int32(time.Date(int(y), time.Month(mo+2), 0, 0, 0, 0, 0, time.UTC).Day()) }

func wrap(v, n int32) int32 { return ((v % n) + n) % n }

// Roll changes one field and wraps inside its range without touching larger fields.
func (c *Calendar) Roll(field, amount int32) {
	c.norm()
	switch field {
	case CalendarYEAR:
		c.y += amount
	case CalendarMONTH:
		c.mo = wrap(c.mo+amount, 12)
		c.d = min(c.d, daysIn(c.y, c.mo))
	case CalendarDAY_OF_MONTH:
		c.d = wrap(c.d-1+amount, daysIn(c.y, c.mo)) + 1
	case CalendarAM_PM:
		if amount%2 != 0 {
			c.h = (c.h + 12) % 24
		}
	case CalendarHOUR:
		c.h = c.h/12*12 + wrap(c.h%12+amount, 12)
	case CalendarHOUR_OF_DAY:
		c.h = wrap(c.h+amount, 24)
	case CalendarMINUTE:
		c.mi = wrap(c.mi+amount, 60)
	case CalendarSECOND:
		c.s = wrap(c.s+amount, 60)
	default:
		panic(&IllegalArgumentException{Message: "Calendar field " + strconv.Itoa(int(field))})
	}
}

func (c *Calendar) GetMinimum(field int32) int32 {
	if field == CalendarDAY_OF_MONTH || field == CalendarYEAR {
		return 1
	}
	return 0
}

func (c *Calendar) GetMaximum(field int32) int32 {
	switch field {
	case CalendarYEAR:
		return 292278994
	case CalendarMONTH, CalendarHOUR:
		return 11
	case CalendarDAY_OF_MONTH:
		return 31
	case CalendarAM_PM:
		return 1
	case CalendarHOUR_OF_DAY:
		return 23
	}
	return 59
}

func (c *Calendar) GetActualMinimum(field int32) int32 { return c.GetMinimum(field) }

func (c *Calendar) GetActualMaximum(field int32) int32 {
	if field == CalendarDAY_OF_MONTH {
		c.norm()
		return daysIn(c.y, c.mo)
	}
	return c.GetMaximum(field)
}

// GetDisplayNames: long names of a MONTH, DAY_OF_WEEK or AM_PM field (style is ignored).
func (c *Calendar) GetDisplayNames(field, style int32, locale any) *Map {
	m := NewMap()
	switch field {
	case CalendarMONTH:
		for i := 0; i < 12; i++ {
			m.Put(time.Month(i+1).String(), int32(i))
		}
	case CalendarDAY_OF_WEEK:
		for i := 0; i < 7; i++ {
			m.Put(time.Weekday(i).String(), int32(i+1))
		}
	case CalendarAM_PM:
		m.Put("AM", int32(0))
		m.Put("PM", int32(1))
	}
	return m
}

// DateFormatField is DateFormat.Field, Format.Field and AttributedCharacterIterator.Attribute.
type DateFormatField struct {
	name  string
	field int32
}

var (
	DateFormatFieldERA          = &DateFormatField{"era", CalendarERA}
	DateFormatFieldYEAR         = &DateFormatField{"year", CalendarYEAR}
	DateFormatFieldMONTH        = &DateFormatField{"month", CalendarMONTH}
	DateFormatFieldDAY_OF_MONTH = &DateFormatField{"day of month", CalendarDAY_OF_MONTH}
	DateFormatFieldDAY_OF_WEEK  = &DateFormatField{"day of week", CalendarDAY_OF_WEEK}
	DateFormatFieldHOUR_OF_DAY0 = &DateFormatField{"hour of day", CalendarHOUR_OF_DAY}
	DateFormatFieldHOUR_OF_DAY1 = &DateFormatField{"hour of day 1", CalendarHOUR_OF_DAY}
	DateFormatFieldHOUR0        = &DateFormatField{"hour", CalendarHOUR}
	DateFormatFieldHOUR1        = &DateFormatField{"hour 1", CalendarHOUR}
	DateFormatFieldMINUTE       = &DateFormatField{"minute", CalendarMINUTE}
	DateFormatFieldSECOND       = &DateFormatField{"second", CalendarSECOND}
	DateFormatFieldMILLISECOND  = &DateFormatField{"millisecond", CalendarMILLISECOND}
	DateFormatFieldAM_PM        = &DateFormatField{"am pm", CalendarAM_PM}
	DateFormatFieldTIME_ZONE    = &DateFormatField{"time zone", -1}
)

func (f *DateFormatField) GetCalendarField() int32 { return f.field }
func (f *DateFormatField) Equals(o any) bool       { return any(f) == o }
func (f *DateFormatField) String() string          { return f.name }

// FieldPosition: the field is a *DateFormatField or nil (new FieldPosition(0)).
type FieldPosition struct {
	attr       *DateFormatField
	begin, end int32
}

func NewFieldPosition(field any) *FieldPosition {
	f, _ := field.(*DateFormatField)
	return &FieldPosition{attr: f}
}

func (p *FieldPosition) GetFieldAttribute() *DateFormatField { return p.attr }
func (p *FieldPosition) GetBeginIndex() int32                { return p.begin }
func (p *FieldPosition) GetEndIndex() int32                  { return p.end }
func (p *FieldPosition) SetBeginIndex(i int32)               { p.begin = i }
func (p *FieldPosition) SetEndIndex(i int32)                 { p.end = i }

func (p *FieldPosition) Equals(o any) bool {
	q, ok := o.(*FieldPosition)
	return ok && q != nil && *p == *q
}

type DateFormatSymbols struct{}

func (DateFormatSymbols) GetAmPmStrings() []string { return []string{"AM", "PM"} }

type ParseException struct{ Message string }

func (e *ParseException) Error() string { return "ParseException: " + e.Message }

// DateFormat is also SimpleDateFormat: every instance is pattern based.
type DateFormat struct {
	parts   []dfPart
	lenient bool
}

type SimpleDateFormat = DateFormat

type dfPart struct {
	lit    string
	letter byte
	n      int
	field  *DateFormatField
}

var datePatterns = [4]string{"EEEE, MMMM d, y", "MMMM d, y", "MMM d, y", "M/d/yy"}
var timePatterns = [4]string{"h:mm:ss a zzzz", "h:mm:ss a z", "h:mm:ss a", "h:mm a"}

func DateFormatGetDateInstance(style int32, locale ...any) *DateFormat {
	return &DateFormat{parts: parsePattern(datePatterns[style]), lenient: true}
}

func DateFormatGetTimeInstance(style int32, locale ...any) *DateFormat {
	return &DateFormat{parts: parsePattern(timePatterns[style]), lenient: true}
}

var letterFields = map[byte]*DateFormatField{'G': DateFormatFieldERA, 'y': DateFormatFieldYEAR, 'M': DateFormatFieldMONTH,
	'd': DateFormatFieldDAY_OF_MONTH, 'E': DateFormatFieldDAY_OF_WEEK, 'H': DateFormatFieldHOUR_OF_DAY0,
	'k': DateFormatFieldHOUR_OF_DAY1, 'h': DateFormatFieldHOUR1, 'K': DateFormatFieldHOUR0, 'm': DateFormatFieldMINUTE,
	's': DateFormatFieldSECOND, 'S': DateFormatFieldMILLISECOND, 'a': DateFormatFieldAM_PM, 'z': DateFormatFieldTIME_ZONE}

func parsePattern(p string) []dfPart {
	var parts []dfPart
	for i := 0; i < len(p); {
		ch := p[i]
		switch f := letterFields[ch]; {
		case ch == '\'':
			j := strings.IndexByte(p[i+1:], '\'')
			lit := "'"
			if j < 0 {
				j = len(p) - i - 1
			} else if j > 0 {
				lit = p[i+1 : i+1+j]
			}
			parts = append(parts, dfPart{lit: lit})
			i += j + 2
		case f != nil:
			j := i
			for j < len(p) && p[j] == ch {
				j++
			}
			parts = append(parts, dfPart{letter: ch, n: j - i, field: f})
			i = j
		default:
			parts = append(parts, dfPart{lit: string(ch)})
			i++
		}
	}
	return parts
}

func (f *DateFormat) SetLenient(l bool) { f.lenient = l }

func (f *DateFormat) GetDateFormatSymbols() DateFormatSymbols { return DateFormatSymbols{} }

func pad(v, n int) string {
	s := strconv.Itoa(v)
	return strings.Repeat("0", max(0, n-len(s))) + s
}

func (p dfPart) format(t time.Time) string {
	h12 := t.Hour() % 12
	switch p.letter {
	case 0:
		return p.lit
	case 'G':
		return "AD"
	case 'y':
		if p.n == 2 {
			return pad(t.Year()%100, 2)
		}
		return pad(t.Year(), p.n)
	case 'M':
		if p.n >= 4 {
			return t.Month().String()
		} else if p.n == 3 {
			return t.Month().String()[:3]
		}
		return pad(int(t.Month()), p.n)
	case 'd':
		return pad(t.Day(), p.n)
	case 'E':
		if p.n >= 4 {
			return t.Weekday().String()
		}
		return t.Weekday().String()[:3]
	case 'H':
		return pad(t.Hour(), p.n)
	case 'k':
		return pad((t.Hour()+23)%24+1, p.n)
	case 'h':
		return pad((h12+11)%12+1, p.n)
	case 'K':
		return pad(h12, p.n)
	case 'm':
		return pad(t.Minute(), p.n)
	case 's':
		return pad(t.Second(), p.n)
	case 'S':
		return pad(t.Nanosecond()/1e6, p.n)
	case 'a':
		return map[bool]string{false: "AM", true: "PM"}[t.Hour() >= 12]
	}
	z, _ := t.Zone()
	return z
}

func (f *DateFormat) Format(d *Date) string {
	var b strings.Builder
	for _, p := range f.parts {
		b.WriteString(p.format(d.t))
	}
	return b.String()
}

type dfRun struct {
	start, end int32
	attr       *DateFormatField
}

// AttributedCharacterIterator: the formatted text, one run per field or literal.
type AttributedCharacterIterator struct {
	text []uint16
	runs []dfRun
	idx  int32
}

func (f *DateFormat) FormatToCharacterIterator(d *Date) *AttributedCharacterIterator {
	it := &AttributedCharacterIterator{}
	for _, p := range f.parts {
		s := utf16.Encode([]rune(p.format(d.t)))
		it.runs = append(it.runs, dfRun{int32(len(it.text)), int32(len(it.text) + len(s)), p.field})
		it.text = append(it.text, s...)
	}
	return it
}

func (it *AttributedCharacterIterator) Current() uint16 {
	if int(it.idx) >= len(it.text) {
		return CharacterIteratorDONE
	}
	return it.text[it.idx]
}

func (it *AttributedCharacterIterator) First() uint16 { it.idx = 0; return it.Current() }

func (it *AttributedCharacterIterator) Next() uint16 {
	it.idx = min(it.idx+1, int32(len(it.text)))
	return it.Current()
}

func (it *AttributedCharacterIterator) SetIndex(i int32) uint16 { it.idx = i; return it.Current() }

// run is the literal run (merged with its literal neighbours, as Java does) or the field run at the index.
func (it *AttributedCharacterIterator) run() dfRun {
	for i, r := range it.runs {
		if it.idx < r.start || it.idx >= r.end {
			continue
		}
		for r.attr == nil && i > 0 && it.runs[i-1].attr == nil {
			i--
			r.start = it.runs[i].start
		}
		for j := i + 1; r.attr == nil && j < len(it.runs) && it.runs[j].attr == nil; j++ {
			r.end = it.runs[j].end
		}
		return r
	}
	return dfRun{it.idx, it.idx, nil}
}

func (it *AttributedCharacterIterator) GetRunStart() int32 { return it.run().start }
func (it *AttributedCharacterIterator) GetRunLimit() int32 { return it.run().end }

func (it *AttributedCharacterIterator) GetAttributes() *Map {
	m := NewMap()
	if a := it.run().attr; a != nil {
		m.Put(a, a)
	}
	return m
}

// Parse reads the pattern strictly (non-lenient ranges) from the start of s; trailing text is ignored like Java.
func (f *DateFormat) Parse(s string) *Date {
	fail := func() { panic(&ParseException{Message: "Unparseable date: \"" + s + "\""}) }
	year, mon, day, hour, min, sec, ms, pm, h12 := 1970, 1, 1, 0, 0, 0, 0, 0, false
	for _, p := range f.parts {
		if p.letter == 0 {
			if !strings.HasPrefix(s, p.lit) {
				fail()
			}
			s = s[len(p.lit):]
			continue
		}
		name, v := "", 0
		switch {
		case p.letter == 'M' && p.n >= 3, p.letter == 'E', p.letter == 'a', p.letter == 'G', p.letter == 'z':
			i := 0
			for i < len(s) && (s[i] >= 'A' && s[i] <= 'Z' || s[i] >= 'a' && s[i] <= 'z') {
				i++
			}
			name, s = s[:i], s[i:]
			if name == "" {
				fail()
			}
		default:
			i := 0
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			n, err := strconv.Atoi(s[:i])
			if err != nil {
				fail()
			}
			v, s = n, s[i:]
		}
		switch p.letter {
		case 'y':
			if p.n == 2 && v < 100 {
				base := time.Now().Year() - 80
				v += base - base%100
				if v < base {
					v += 100
				}
			}
			year = v
		case 'M':
			mon = v
			if name != "" {
				mon = 0
				for m := 1; m <= 12; m++ {
					full := time.Month(m).String()
					if strings.EqualFold(name, full) || strings.EqualFold(name, full[:3]) {
						mon = m
					}
				}
				if mon == 0 {
					fail()
				}
			}
		case 'd':
			day = v
		case 'H':
			hour = v
		case 'k':
			hour = v % 24
		case 'h', 'K':
			hour, h12 = v%12, true
		case 'm':
			min = v
		case 's':
			sec = v
		case 'S':
			ms = v
		case 'a':
			switch strings.ToUpper(name) {
			case "AM":
			case "PM":
				pm = 1
			default:
				fail()
			}
		}
	}
	if h12 {
		hour += 12 * pm
	}
	t := time.Date(year, time.Month(mon), day, hour, min, sec, ms*1e6, time.Local)
	if !f.lenient && (mon < 1 || mon > 12 || day != t.Day() || hour > 23 || min > 59 || sec > 59) {
		fail()
	}
	return &Date{t}
}

// Collections holds the static helpers the translated sources call: max, emptySet, singleton.
type Collections struct{}

func CollectionsEmptySet() *List { return NewList() }

func CollectionsSingleton(v any) *List { return ListOf(v) }

func CollectionsMax[T any](c *List, cmp func(T, T) int32) T {
	best := c.items[0].(T)
	for _, v := range c.items[1:] {
		if cmp(v.(T), best) > 0 {
			best = v.(T)
		}
	}
	return best
}
