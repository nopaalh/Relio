package models_test

import (
	"github.com/nopaalh/Relio/backend/models"
	"testing"
	"time"
)

func TestDateParsingStrict(t *testing.T) {
	for _, raw := range []string{"2026-10-01", "2024-02-29", "2027-01-01"} {
		got, err := models.ParseDate(raw)
		if err != nil || string(got) != raw {
			t.Fatalf("ParseDate(%q) = %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"", "2026-02-29", "2026-02-30", "2026-13-01", "2026-1-01", " 2026-10-01", "2026-10-01 ", "2026-10-01T00:00:00Z", "0000-01-01"} {
		if _, err := models.ParseDate(raw); err == nil {
			t.Fatalf("accepted invalid calendar token %q", raw)
		}
	}
}

func TestDateNextDayUsesCalendarAndPreservesRaw(t *testing.T) {
	for _, tt := range []struct {
		raw  models.Date
		want models.Date
	}{
		{"2026-08-15", "2026-08-16"}, // team convention for last inclusive employment day
		{"2026-12-31", "2027-01-01"}, {"2024-02-28", "2024-02-29"}, {"2024-02-29", "2024-03-01"},
	} {
		raw := tt.raw
		got, err := raw.NextDay()
		if err != nil || got != tt.want || raw != tt.raw {
			t.Fatalf("NextDay(%q) = %q, %v", raw, got, err)
		}
	}
	for _, raw := range []models.Date{"", "not-date", "9999-12-31"} {
		if _, err := raw.NextDay(); err == nil {
			t.Fatalf("NextDay accepted %q", raw)
		}
	}
}

func TestDateNextDayStartIncludesEntireJakartaDay(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	cutoff, err := models.Date("2026-10-01").NextDayStart(jakarta)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC)
	if !cutoff.Equal(want) {
		t.Fatalf("exclusive bound = %v, want %v", cutoff, want)
	}
	last := time.Date(2026, 10, 1, 23, 59, 59, 0, jakarta)
	if !last.Before(cutoff) {
		t.Fatal("last second of as_of excluded")
	}
	if time.Date(2026, 10, 2, 0, 0, 0, 0, jakarta).Before(cutoff) {
		t.Fatal("future next-day midnight included")
	}
	if _, err := models.Date("2026-10-01").NextDayStart(nil); err == nil {
		t.Fatal("nil calendar zone accepted")
	}
	if _, err := models.Date("bad").NextDayStart(jakarta); err == nil {
		t.Fatal("invalid date accepted")
	}
}
