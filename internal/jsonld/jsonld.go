// Package jsonld parses schema.org Restaurant JSON-LD blocks embedded in
// Find Me Gluten Free /biz/* pages. Each biz page contains exactly one
// <script type="application/ld+json">{...}</script> block. The block is
// JSON, but its string values contain HTML entities (&#039;, &amp;, etc.)
// that must be decoded after json.Unmarshal.
package jsonld

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Restaurant is the parsed schema.org Restaurant payload from a biz page.
// All string fields have HTML entities decoded.
type Restaurant struct {
	Name        string    `json:"name,omitempty"`
	Image       string    `json:"image,omitempty"`
	PriceRange  string    `json:"price_range,omitempty"`
	Telephone   string    `json:"telephone,omitempty"`
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	RatingValue float64   `json:"rating_value,omitempty"`
	RatingCount int       `json:"rating_count,omitempty"`
	Reviews     []Review  `json:"reviews,omitempty"`
	BizID       string    `json:"biz_id,omitempty"`
	Slug        string    `json:"slug,omitempty"`
	URL         string    `json:"url,omitempty"`
	City        string    `json:"city,omitempty"`
	State       string    `json:"state,omitempty"`
	Country     string    `json:"country,omitempty"`
	Address     string    `json:"address,omitempty"`
	OGTitle     string    `json:"og_title,omitempty"`
	OGDesc      string    `json:"og_description,omitempty"`
	HasGFMenu   bool      `json:"has_gf_menu,omitempty"`
	Dedicated   bool      `json:"dedicated,omitempty"`
	FetchedAt   time.Time `json:"fetched_at,omitempty"`
}

// Review is one user review with HTML-entity-decoded text.
type Review struct {
	Rating        float64 `json:"rating,omitempty"`
	DatePublished string  `json:"date_published,omitempty"`
	Description   string  `json:"description,omitempty"`
	Author        string  `json:"author,omitempty"`
}

// rawRestaurant mirrors the on-page JSON-LD shape just enough to decode it.
type rawRestaurant struct {
	Context    string              `json:"@context"`
	Type       string              `json:"@type"`
	Name       string              `json:"name"`
	Image      string              `json:"image"`
	PriceRange string              `json:"priceRange"`
	Telephone  string              `json:"telephone"`
	Location   *rawLocation        `json:"location"`
	AggRating  *rawAggregateRating `json:"aggregateRating"`
	Review     []rawReview         `json:"review"`
}

type rawLocation struct {
	Type string  `json:"@type"`
	Geo  *rawGeo `json:"geo"`
}

type rawGeo struct {
	Type      string `json:"@type"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type rawAggregateRating struct {
	Type        string  `json:"@type"`
	RatingValue float64 `json:"ratingValue"`
	RatingCount int     `json:"ratingCount"`
}

type rawReview struct {
	ReviewRating  *rawReviewRating `json:"reviewRating"`
	DatePublished string           `json:"datePublished"`
	Description   string           `json:"description"`
	Author        *rawAuthor       `json:"author"`
}

type rawReviewRating struct {
	RatingValue float64 `json:"ratingValue"`
}

type rawAuthor struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

// jsonldRegex finds the JSON content inside the first
// <script type="application/ld+json">...</script> block.
// The script tag attribute order can vary slightly; we match the type
// attribute and grab the body.
var jsonldRegex = regexp.MustCompile(`(?s)<script\s+type=["']application/ld\+json["']\s*>(.*?)</script>`)

// ogPropertyRegex matches <meta property="og:KEY" content="VALUE"/>.
var ogPropertyRegex = regexp.MustCompile(`<meta\s+property=["']og:([^"']+)["']\s+content=["']([^"']*)["']`)

// canonicalAddressRegex extracts a city, state hint from og:title in the
// shape "Pinky's - Gluten-Free Restaurant in Richmond, VA". This is a
// best-effort fallback; the structured location.geo provides the real coords.
var canonicalAddressRegex = regexp.MustCompile(`in\s+([A-Za-z][\w\s\.\-']+?),\s+([A-Z]{2,3})$`)

// urlBizRegex extracts country/state/city from a /biz canonical URL when the
// page surfaces one. FMGF biz canonicals do not always carry city — the
// city is always present in og:title prose.
var urlBizRegex = regexp.MustCompile(`/biz/([^/]+)/(\d+)`)

// Parse extracts the schema.org Restaurant from raw biz-page HTML.
// Returns an error only on a hard parse failure (no JSON-LD block, or
// invalid JSON). Missing fields within the block produce a Restaurant
// with zero values for those fields, not an error — the page can be
// partially populated, especially for newly listed places.
func Parse(htmlBody []byte, fetchURL string) (*Restaurant, error) {
	body := string(htmlBody)
	match := jsonldRegex.FindStringSubmatch(body)
	if match == nil {
		return nil, fmt.Errorf("no application/ld+json block found")
	}
	jsonBody := normalizeJSONStringLiterals(strings.TrimSpace(match[1]))

	var raw rawRestaurant
	if err := json.Unmarshal([]byte(jsonBody), &raw); err != nil {
		return nil, fmt.Errorf("decoding JSON-LD: %w", err)
	}

	r := &Restaurant{
		Name:       html.UnescapeString(raw.Name),
		Image:      raw.Image,
		PriceRange: html.UnescapeString(raw.PriceRange),
		Telephone:  raw.Telephone,
		URL:        fetchURL,
		FetchedAt:  time.Now().UTC(),
	}

	if raw.Location != nil && raw.Location.Geo != nil {
		if lat, err := strconv.ParseFloat(raw.Location.Geo.Latitude, 64); err == nil {
			r.Latitude = lat
		}
		if lng, err := strconv.ParseFloat(raw.Location.Geo.Longitude, 64); err == nil {
			r.Longitude = lng
		}
	}

	if raw.AggRating != nil {
		r.RatingValue = raw.AggRating.RatingValue
		r.RatingCount = raw.AggRating.RatingCount
	}

	for _, rv := range raw.Review {
		review := Review{
			DatePublished: rv.DatePublished,
			Description:   html.UnescapeString(rv.Description),
		}
		if rv.ReviewRating != nil {
			review.Rating = rv.ReviewRating.RatingValue
		}
		if rv.Author != nil {
			review.Author = html.UnescapeString(rv.Author.Name)
		}
		r.Reviews = append(r.Reviews, review)
	}

	// OG metadata adds the city/state context the JSON-LD omits.
	for _, m := range ogPropertyRegex.FindAllStringSubmatch(body, -1) {
		key, val := m[1], html.UnescapeString(m[2])
		switch key {
		case "title":
			r.OGTitle = val
			if cm := canonicalAddressRegex.FindStringSubmatch(val); cm != nil {
				r.City = strings.TrimSpace(cm[1])
				r.State = strings.TrimSpace(cm[2])
			}
		case "description":
			r.OGDesc = val
		}
	}

	// Detect dedicated / GF-menu flags from the OG description prose. FMGF
	// uses near-stable phrasings: "Dedicated gluten-free" vs "not dedicated
	// gluten-free but has a gluten-free menu". This is heuristic; reviews
	// override when in doubt.
	desc := strings.ToLower(r.OGDesc)
	switch {
	case strings.Contains(desc, "dedicated gluten-free") && !strings.Contains(desc, "not dedicated"):
		r.Dedicated = true
		r.HasGFMenu = true
	case strings.Contains(desc, "gluten-free menu"):
		r.HasGFMenu = true
	}

	// Extract slug + biz_id from the canonical URL the caller passed.
	if um := urlBizRegex.FindStringSubmatch(fetchURL); um != nil {
		r.Slug = um[1]
		r.BizID = um[2]
	}

	return r, nil
}

// MarshalIndent returns the Restaurant as pretty JSON.
func (r *Restaurant) MarshalIndent() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// normalizeJSONStringLiterals walks the JSON byte stream and escapes
// literal control characters (newline, carriage return, tab) that appear
// inside string values. The Find Me Gluten Free backend emits review
// `description` fields with raw newlines embedded — strict JSON forbids
// this, so json.Unmarshal returns "invalid character '\n' in string
// literal". Rewriting the bytes before decoding produces a well-formed
// JSON document with the same logical content.
//
// The walker is a tiny string-state machine: outside a string, bytes pass
// through untouched; inside a string, control chars are escape-encoded
// while the backslash-escape sequence is preserved verbatim.
func normalizeJSONStringLiterals(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !inString {
			b.WriteByte(c)
			if c == '"' {
				inString = true
			}
			continue
		}
		if escape {
			b.WriteByte(c)
			escape = false
			continue
		}
		switch c {
		case '\\':
			b.WriteByte(c)
			escape = true
		case '"':
			b.WriteByte(c)
			inString = false
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
