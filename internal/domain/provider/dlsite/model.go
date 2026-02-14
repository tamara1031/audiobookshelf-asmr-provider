package dlsite

import (
	"errors"
	"regexp"
	"strings"
)

// RJCode represents a DLsite work ID (e.g., RJ123456).
type RJCode struct {
	value string
}

var rjCodeRegex = regexp.MustCompile(`(?i)^RJ\d{6,8}$`)

// NewRJCode validates and creates a new RJCode.
func NewRJCode(code string) (RJCode, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !rjCodeRegex.MatchString(code) {
		return RJCode{}, errors.New("invalid RJ code format")
	}
	return RJCode{value: code}, nil
}

// String returns the string representation of the RJCode.
func (r RJCode) String() string {
	return r.value
}

// AsmrWork represents the DLsite-specific entity for an ASMR work.
type AsmrWork struct {
	RJCode      RJCode
	Title       string
	Circle      string
	CV          []string
	Tags        []string
	Description string
	CoverURL    string
	Price       int
	ReleaseDate string
	DLsiteURL   string
	Series      string
	Scenario    string
	WorkFormat  string
	AgeRating   string
}
