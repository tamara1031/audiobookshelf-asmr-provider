package domain

import (
	"context"
	"time"
)

// AbsMetadataResponse represents the top-level JSON response expected by Audiobookshelf.
type AbsMetadataResponse struct {
	Matches []AbsBookMetadata `json:"matches"`
}

// AbsBookMetadata matches the JSON structure for a book/work in the ABS custom provider API.
type AbsBookMetadata struct {
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle,omitempty"`
	Author        string   `json:"author"`
	Narrator      string   `json:"narrator,omitempty"`
	Series        string   `json:"series,omitempty"`
	Description   string   `json:"description,omitempty"`
	Publisher     string   `json:"publisher,omitempty"`
	PublishedYear string   `json:"publishedYear,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Cover         string   `json:"cover,omitempty"`
	ISBN          string   `json:"isbn,omitempty"`
	ASIN          string   `json:"asin,omitempty"`
	Language      string   `json:"language,omitempty"`
	Explicit      bool     `json:"explicit,omitempty"`
}

// Provider defines the interface for a metadata provider plugin.
type Provider interface {
	// ID returns the unique identifier of the provider (e.g., "dlsite").
	ID() string

	// Search searches for works matching the query and returns ABS-compatible metadata.
	Search(ctx context.Context, query string) ([]AbsBookMetadata, error)

	// CacheTTL returns the duration for which results should be cached.
	CacheTTL() time.Duration
}
