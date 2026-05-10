// Package safety identifies celiac safety reports inside a review's text
// using a curated keyword list. The list is deterministic, reviewable, and
// no LLM is involved — agents and users can reason about why a review
// matched. The list is intentionally narrow: false positives mute users
// to safety signals; false negatives just mean the user reads more.
//
// pp:novel-static-reference — this list is curated from celiac community
// vernacular about cross-contamination and gluten exposure. It is not
// exhaustive; expand only when a real review is found that should match
// but doesn't.
package safety

import "strings"

// Keywords is the canonical set of phrases that mark a review as a
// celiac-safety report. Match is case-insensitive and substring-based.
var Keywords = []string{
	// Direct exposure verbs / nouns
	"got glutened",
	"glutened",
	"got sick",
	"made me sick",
	"reacted",
	"had a reaction",
	"celiac reaction",
	"i was glutened",

	// Cross-contamination phrases
	"cross-contamination",
	"cross contamination",
	"cross-contam",
	"cross contam",
	"shared fryer",
	"not a dedicated fryer",
	"contaminated",
	"contamination",

	// Symptom phrases (specific to celiac/gluten exposure)
	"glutening",
	"flour everywhere",
	"flour in the kitchen",
	"flour dust",
	"breaded together",
	"same surface",
}

// IsSafetyReport returns true when description contains any safety keyword.
// Match is case-insensitive substring; Description is normalized to lower
// once before checks to keep cost O(N*M) over the keyword list.
func IsSafetyReport(description string) bool {
	if description == "" {
		return false
	}
	d := strings.ToLower(description)
	for _, k := range Keywords {
		if strings.Contains(d, k) {
			return true
		}
	}
	return false
}
