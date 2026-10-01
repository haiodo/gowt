package jrt

import "testing"

func TestDateFormatRoundTrip(t *testing.T) {
	c := CalendarGetInstance()
	c.Set(2024, 0, 31)
	c.Set(CalendarHOUR_OF_DAY, 15)
	c.Set(CalendarMINUTE, 4)
	c.Set(CalendarSECOND, 9)
	f := DateFormatGetDateInstance(DateFormatMEDIUM)
	if got := f.Format(c.GetTime()); got != "Jan 31, 2024" {
		t.Fatal(got)
	}
	if got := DateFormatGetTimeInstance(DateFormatMEDIUM).Format(c.GetTime()); got != "3:04:09 PM" {
		t.Fatal(got)
	}
	c.Roll(CalendarMONTH, 1) // the day clamps to the month length
	if c.Get(CalendarMONTH) != 1 || c.Get(CalendarDAY_OF_MONTH) != 29 {
		t.Fatal(c.Get(CalendarMONTH), c.Get(CalendarDAY_OF_MONTH))
	}
	f.SetLenient(false)
	if d := f.Parse("Mar 5, 2021"); DateFormatGetDateInstance(DateFormatSHORT).Format(d) != "3/5/21" {
		t.Fatal("parse")
	}
	it := f.FormatToCharacterIterator(c.GetTime())
	if it.GetRunLimit() != 3 || it.GetAttributes().Get(DateFormatFieldMONTH) == nil {
		t.Fatal("first run is the month")
	}
}
