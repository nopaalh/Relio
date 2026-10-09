package repository_test

import (
	"errors"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"testing"
)

func TestGraphBoundsDefaultsAndLimits(t *testing.T) {
	got, err := repository.NormalizeGraphOptions(models.GraphOptions{})
	if err != nil || got.Depth != 1 || got.MaxNodes != 150 || got.MaxEdges != 300 {
		t.Fatalf("defaults = %+v, %v", got, err)
	}
	focus := "TEST-EVENT"
	input := models.GraphOptions{Depth: 2, MaxNodes: 10, MaxEdges: 20, FocusEventID: &focus}
	got, err = repository.NormalizeGraphOptions(input)
	if err != nil || got != input {
		t.Fatalf("valid bounds changed = %+v, %v", got, err)
	}
}

func TestGraphBoundsRejectInvalidWithoutClamp(t *testing.T) {
	empty, blank := "", " "
	for _, input := range []models.GraphOptions{
		{Depth: -1}, {Depth: 3}, {MaxNodes: -1}, {MaxNodes: 151}, {MaxEdges: -1}, {MaxEdges: 301},
		{FocusEventID: &empty}, {FocusEventID: &blank},
	} {
		_, err := repository.NormalizeGraphOptions(input)
		var typed *repository.RepositoryError
		if !errors.As(err, &typed) || typed.Code != repository.InvalidQuery {
			t.Fatalf("invalid bounds accepted: %+v, %v", input, err)
		}
	}
}

func TestPageLimitDefaultsAndBounds(t *testing.T) {
	for _, tt := range []struct{ input, want int }{{0, 50}, {1, 1}, {50, 50}} {
		got, err := repository.NormalizePageLimit(tt.input)
		if err != nil || got != tt.want {
			t.Fatalf("limit %d = %d, %v", tt.input, got, err)
		}
	}
	for _, limit := range []int{-1, 51, 1000} {
		var typed *repository.RepositoryError
		if _, err := repository.NormalizePageLimit(limit); !errors.As(err, &typed) || typed.Code != repository.InvalidQuery {
			t.Fatalf("invalid limit %d = %v", limit, err)
		}
	}
}
