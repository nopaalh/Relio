package repository

import (
	"github.com/nopaalh/Relio/backend/models"
	"strings"
)

// NormalizeGraphOptions applies absent-value defaults, never clamps invalid
// values. Focus ownership, visibility, and traversal enforcement are adapter work.
func NormalizeGraphOptions(options models.GraphOptions) (models.GraphOptions, error) {
	if options.Depth == 0 {
		options.Depth = 1
	}
	if options.MaxNodes == 0 {
		options.MaxNodes = 150
	}
	if options.MaxEdges == 0 {
		options.MaxEdges = 300
	}
	if options.Depth < 1 || options.Depth > 2 || options.MaxNodes < 1 || options.MaxNodes > 150 || options.MaxEdges < 1 || options.MaxEdges > 300 {
		return models.GraphOptions{}, &RepositoryError{Code: InvalidQuery}
	}
	if options.FocusEventID != nil && strings.TrimSpace(*options.FocusEventID) == "" {
		return models.GraphOptions{}, &RepositoryError{Code: InvalidQuery}
	}
	return options, nil
}

func NormalizePageLimit(limit int) (int, error) {
	if limit == 0 {
		return 50, nil
	}
	if limit < 1 || limit > 50 {
		return 0, &RepositoryError{Code: InvalidQuery}
	}
	return limit, nil
}
