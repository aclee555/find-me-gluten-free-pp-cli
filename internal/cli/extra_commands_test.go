package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/aclee555/find-me-gluten-free-pp-cli/internal/jsonld"
	"github.com/aclee555/find-me-gluten-free-pp-cli/internal/store"
)

// Reproduces the bug Rufio filed: after `cities sync us ca los-angeles`
// the city_snapshot row holds the authoritative biz_ids for the slug,
// but downstream commands resolved membership by string-matching the
// per-biz City field. When the JSON-LD og:title regex didn't populate
// City — which happens whenever the title doesn't end exactly with
// "in <City>, <ST>" — the membership lookup found nothing and
// places stats / reviews recent returned zero results despite the
// hydrated rows being right there.
//
// The fix: prefer the snapshot when one exists.
func TestLoadCityBizesFromStore_PrefersSnapshotOverCityField(t *testing.T) {
	s := newTempStore(t)
	defer s.Close()

	// Two LA bizes hydrated by `cities sync` — neither has a usable
	// City field because the og:title shape on the live page didn't
	// match the canonicalAddressRegex.
	mustUpsertBiz(t, s, jsonld.Restaurant{
		BizID: "111", Name: "KUKU Cafe",
		Latitude: 33.96757, Longitude: -118.354225,
		RatingValue: 5, RatingCount: 3,
		Reviews: []jsonld.Review{{Rating: 1, DatePublished: "2026-04-01", Description: "got glutened"}},
	})
	mustUpsertBiz(t, s, jsonld.Restaurant{
		BizID: "222", Name: "Modern Bread and Bagel",
		RatingValue: 5, RatingCount: 38,
	})
	// An unrelated biz the snapshot does not include.
	mustUpsertBiz(t, s, jsonld.Restaurant{BizID: "999", Name: "Elsewhere", City: "Portland"})

	mustUpsertSnapshot(t, s, citySnapshot{
		CountryState: "us/ca",
		City:         "los-angeles",
		BizIDs:       []string{"111", "222"},
		TakenAt:      time.Date(2026, 5, 10, 21, 29, 41, 0, time.UTC),
	})

	got, err := loadCityBizesFromStore(s, "los-angeles")
	if err != nil {
		t.Fatalf("loadCityBizesFromStore: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 LA bizes from snapshot, got %d (%+v)", len(got), got)
	}
	stats := computeCityStats("los-angeles", got, false)
	if stats.Total != 2 {
		t.Errorf("stats.Total: want 2, got %d", stats.Total)
	}
	if stats.TotalReviews != 41 {
		t.Errorf("stats.TotalReviews: want 41 (3+38), got %d", stats.TotalReviews)
	}
}

// When no snapshot exists for the slug, fall back to City-substring
// matching so single-biz `places hydrate` flows still work.
func TestLoadCityBizesFromStore_FallsBackToCityField(t *testing.T) {
	s := newTempStore(t)
	defer s.Close()

	mustUpsertBiz(t, s, jsonld.Restaurant{BizID: "1", Name: "Pinky's", City: "Richmond"})
	mustUpsertBiz(t, s, jsonld.Restaurant{BizID: "2", Name: "Elsewhere", City: "Portland"})

	got, err := loadCityBizesFromStore(s, "richmond")
	if err != nil {
		t.Fatalf("loadCityBizesFromStore: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Pinky's" {
		t.Fatalf("want 1 Richmond biz (Pinky's), got %+v", got)
	}
}

// Reproduces the `cities saved --agent` returning [{}] bug. The agent
// pipeline routes through compactListFields, whose generic allowlist
// (id/name/title/status/...) does not include any field on the
// savedCityEntry struct (city_slug/added_at/note). Before the fix every
// item became {}; after the fix items with no allowlisted fields are
// returned whole so domain resources stay readable.
func TestCompactListFields_PreservesDomainOnlyItems(t *testing.T) {
	items := []map[string]any{
		{"city_slug": "los-angeles", "added_at": "2026-05-10T21:30:43Z", "note": "family trip"},
	}
	out := compactListFields(items)
	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 item, got %d", len(got))
	}
	if got[0]["city_slug"] != "los-angeles" {
		t.Errorf("city_slug not preserved: %+v", got[0])
	}
	if got[0]["note"] != "family trip" {
		t.Errorf("note not preserved: %+v", got[0])
	}
}

// Items that DO have allowlisted fields keep being shortened — the
// fallback only kicks in when the filtered object would be empty.
func TestCompactListFields_ShortensItemsWithAllowlistedFields(t *testing.T) {
	items := []map[string]any{
		{"id": "abc", "name": "Project X", "description": "very long body...", "comments": []any{"a", "b"}},
	}
	out := compactListFields(items)
	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, has := got[0]["description"]; has {
		t.Errorf("description should have been stripped: %+v", got[0])
	}
	if got[0]["id"] != "abc" || got[0]["name"] != "Project X" {
		t.Errorf("expected id+name kept: %+v", got[0])
	}
}

func newTempStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return s
}

func mustUpsertBiz(t *testing.T, s *store.Store, r jsonld.Restaurant) {
	t.Helper()
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("marshal biz: %v", err)
	}
	if err := s.Upsert(resTypeBiz, storeID(resTypeBiz, r.BizID), data); err != nil {
		t.Fatalf("upsert biz: %v", err)
	}
}

func mustUpsertSnapshot(t *testing.T, s *store.Store, snap citySnapshot) {
	t.Helper()
	data, err := json.Marshal(&snap)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if err := s.Upsert(resTypeSnapshot, storeID(resTypeSnapshot, snap.id()), data); err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}
}
