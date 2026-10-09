package models

import (
	"errors"
	"time"
	_ "time/tzdata" // Keep Asia/Jakarta available on Windows without external zone files.
)

// ParseDate validates a calendar token without trimming, clamping, or
// accepting timestamps. Planned dates may exceed the dataset snapshot;
// the snapshot guard, not this general date parser, enforces the cutoff.
func ParseDate(raw string) (Date, error) {
	if len(raw) != 10 {
		return "", errors.New("invalid calendar date")
	}
	parsed, err := time.Parse(time.DateOnly, raw)
	if err != nil || parsed.Year() < 1 || parsed.Format(time.DateOnly) != raw {
		return "", errors.New("invalid calendar date")
	}
	return Date(raw), nil
}

func (d Date) NextDay() (Date, error) {
	if _, err := ParseDate(string(d)); err != nil {
		return "", err
	}
	parsed, _ := time.Parse(time.DateOnly, string(d))
	return ParseDate(parsed.AddDate(0, 0, 1).Format(time.DateOnly))
}

// NextDayStart is an exclusive query boundary, not a source event timestamp.
func (d Date) NextDayStart(zone *time.Location) (time.Time, error) {
	if zone == nil {
		return time.Time{}, errors.New("calendar zone required")
	}
	next, err := d.NextDay()
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(time.DateOnly, string(next), zone)
}
