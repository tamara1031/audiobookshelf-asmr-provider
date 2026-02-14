package service

import (
	"context"
	"time"
)

// AbsBookMetadata represents the metadata structure used by Audiobookshelf.
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

// AbsMetadataResponse represents the search response format for Audiobookshelf.
type AbsMetadataResponse struct {
	Matches []AbsBookMetadata `json:"matches"`
}

// Provider defines the interface for all metadata provider plugins.
type Provider interface {
	// ID returns the unique identifier of the provider (e.g., "dlsite").
	ID() string

	// Search searches for works matching the query and returns ABS-compatible metadata.
	Search(ctx context.Context, query string) ([]AbsBookMetadata, error)

	// CacheTTL returns the duration for which results should be cached.
	CacheTTL() time.Duration
}
