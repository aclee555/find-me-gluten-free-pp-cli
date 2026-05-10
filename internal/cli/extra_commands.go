package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aclee555/find-me-gluten-free-pp-cli/internal/client"
	"github.com/aclee555/find-me-gluten-free-pp-cli/internal/jsonld"
	"github.com/aclee555/find-me-gluten-free-pp-cli/internal/safety"
	"github.com/aclee555/find-me-gluten-free-pp-cli/internal/store"
)

// bizLinkRegex matches href="/biz/<slug>/<id>" anchors (id is digits).
var bizLinkRegex = regexp.MustCompile(`href="/biz/([^"/]+)/(\d+)"`)

// wireExtraCommands attaches the 8 transcendence commands and the
// quality-of-life absorbed commands (bookmarks, saved cities, postal
// lookup, place hydration, place detail) onto the root command tree.
//
// Why this lives in its own file: root.go is generated and re-emitted on
// every regen. Adding our novel features here keeps the regen-merge clean
// — root.go gains a single line (`wireExtraCommands(rootCmd, flags)`) and
// every new command lands in this file.
func wireExtraCommands(rootCmd *cobra.Command, flags *rootFlags) {
	// Place subcommands attach to the existing `places` parent.
	if placesCmd, _, err := rootCmd.Find([]string{"places"}); err == nil && placesCmd != nil {
		placesCmd.AddCommand(newPlacesHydrateCmd(flags))
		placesCmd.AddCommand(newPlacesDetailCmd(flags))
		placesCmd.AddCommand(newPlacesNearCmd(flags))
		placesCmd.AddCommand(newPlacesStatsCmd(flags))
		placesCmd.AddCommand(newPlacesPostalCmd(flags))
	}
	// Cities gains diff + a sync subcommand.
	if citiesCmd, _, err := rootCmd.Find([]string{"cities"}); err == nil && citiesCmd != nil {
		citiesCmd.AddCommand(newCitiesDiffCmd(flags))
		citiesCmd.AddCommand(newCitiesSyncCmd(flags))
		citiesCmd.AddCommand(newCitiesSaveCmd(flags))
		citiesCmd.AddCommand(newCitiesUnsaveCmd(flags))
		citiesCmd.AddCommand(newCitiesSavedCmd(flags))
	}

	// Top-level new resources.
	rootCmd.AddCommand(newTripCmd(flags))
	rootCmd.AddCommand(newReviewsCmd(flags))
	rootCmd.AddCommand(newWatchCmd(flags))
	rootCmd.AddCommand(newCuisinesCmd(flags))
	rootCmd.AddCommand(newBookmarkCmd(flags))
}

// ============================================================
// User-Agent etiquette
// ============================================================

// fmgfUserAgent is the honest, identifying User-Agent the CLI sends on
// every HTTP request. Find Me Gluten Free's robots.txt extra-disallows
// AI bots from /biz, /posts, /postal — we make our identity transparent
// rather than impersonate Chrome.
const fmgfUserAgent = "github.com/aclee555/find-me-gluten-free-pp-cli/0.1.0 (+https://github.com/mvanhorn/find-me-gluten-free-pp-cli)"

// ============================================================
// Storage helpers (resource_type strings)
// ============================================================

const (
	resTypeBiz       = "biz"
	resTypeSnapshot  = "city_snapshot"
	resTypeBookmark  = "bookmark"
	resTypeSavedCity = "saved_city"
	resTypeWatch     = "watch"
)

// The generated resources table uses id as PRIMARY KEY (not the
// (resource_type, id) composite). Multiple resource types in this CLI
// naturally share the same domain IDs — a biz row and the bookmark for
// that biz are both keyed on the biz_id — so we namespace IDs at the
// store boundary to keep them from colliding via ON CONFLICT(id).
func storeID(resourceType, id string) string {
	return resourceType + ":" + id
}

func openStore(flags *rootFlags) (*store.Store, error) {
	return store.Open(defaultDBPath("find-me-gluten-free-pp-cli"))
}

// ============================================================
// places hydrate <slug> <biz_id>
// ============================================================

func newPlacesHydrateCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hydrate [slug] [biz_id]",
		Short: "Fetch a restaurant page, parse its schema.org JSON-LD, and cache it locally",
		Long: `Fetch a single restaurant from Find Me Gluten Free, parse the embedded
schema.org JSON-LD (rating, reviews, lat/lng, telephone, price), and store
it in the local SQLite cache so subsequent commands work offline.

Foundation for trip compare, places near, places stats, reviews recent, and
all the other novel features.`,
		Example:     "  find-me-gluten-free-pp-cli places hydrate pinkys 6118524171452416 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return usageErr(fmt.Errorf("usage: places hydrate <slug> <biz_id>"))
			}
			slug, bizID := args[0], args[1]
			if dryRunOK(flags) {
				fmt.Fprintf(cmd.OutOrStdout(), "would hydrate /biz/%s/%s\n", slug, bizID)
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			r, err := hydrateOne(cmd.Context(), c, slug, bizID)
			if err != nil {
				return err
			}
			if err := persistBiz(flags, r); err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), r, flags)
		},
	}
	return cmd
}

// hydrateOne fetches one biz page with the honest User-Agent and parses JSON-LD.
func hydrateOne(ctx context.Context, c *client.Client, slug, bizID string) (*jsonld.Restaurant, error) {
	path := fmt.Sprintf("/biz/%s/%s", slug, bizID)
	body, err := getHTML(ctx, c, path)
	if err != nil {
		return nil, fmt.Errorf("fetching biz page: %w", err)
	}
	r, err := jsonld.Parse(body, c.BaseURL+path)
	if err != nil {
		return nil, fmt.Errorf("parsing biz JSON-LD: %w", err)
	}
	return r, nil
}

// getHTML uses the underlying HTTP client with our honest User-Agent
// override. The generated client.Get path goes through Surf with Chrome
// impersonation; for personal-CLI etiquette we send identifying headers.
func getHTML(ctx context.Context, c *client.Client, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmgfUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		return nil, rateLimitErr(fmt.Errorf("HTTP 429 from findmeglutenfree.com — lower the rate (try export FMGF_RPS=0.5)"))
	}
	if resp.StatusCode >= 400 {
		return nil, &client.APIError{Method: "GET", Path: path, StatusCode: resp.StatusCode}
	}
	const maxBody = 4 << 20 // 4 MB cap
	buf := make([]byte, 0, 64<<10)
	tmp := make([]byte, 32<<10)
	for {
		n, rerr := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > maxBody {
				return nil, fmt.Errorf("response exceeded %d bytes", maxBody)
			}
		}
		if rerr != nil {
			break
		}
	}
	return buf, nil
}

func persistBiz(flags *rootFlags, r *jsonld.Restaurant) error {
	s, err := openStore(flags)
	if err != nil {
		return err
	}
	defer s.Close()
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return s.Upsert(resTypeBiz, storeID(resTypeBiz, r.BizID), data)
}

// ============================================================
// places detail <biz_id> — read enriched data from local cache
// ============================================================

func newPlacesDetailCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "detail [biz_id]",
		Short:       "Read a hydrated restaurant from the local cache (rich rating, reviews, lat/lng)",
		Example:     "  find-me-gluten-free-pp-cli places detail 6118524171452416 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			r, err := loadBiz(flags, args[0])
			if err != nil {
				return notFoundErr(fmt.Errorf("biz %s not in local cache; run: places hydrate <slug> %s", args[0], args[0]))
			}
			return printJSONFiltered(cmd.OutOrStdout(), r, flags)
		},
	}
	return cmd
}

func loadBiz(flags *rootFlags, bizID string) (*jsonld.Restaurant, error) {
	s, err := openStore(flags)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	raw, err := s.Get(resTypeBiz, storeID(resTypeBiz, bizID))
	if err != nil {
		return nil, err
	}
	var r jsonld.Restaurant
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ============================================================
// places near --lat --lng --radius-km
// ============================================================

type nearResult struct {
	BizID      string  `json:"biz_id"`
	Name       string  `json:"name"`
	Slug       string  `json:"slug"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	DistanceKm float64 `json:"distance_km"`
	Rating     float64 `json:"rating,omitempty"`
	Dedicated  bool    `json:"dedicated"`
	HasGFMenu  bool    `json:"has_gf_menu"`
}

func newPlacesNearCmd(flags *rootFlags) *cobra.Command {
	var lat, lng, radius float64
	var dedicated, hasGFMenu bool
	var minRating float64
	var limit int
	cmd := &cobra.Command{
		Use:   "near",
		Short: "Find dedicated-GF or GF-menu places within a radius of a lat/lng (Haversine over local cache)",
		Long: `Searches the locally cached, hydrated restaurants for places within
--radius-km of the provided lat/lng. Combines with --dedicated, --has-gf-menu,
and --min-rating filters. Results are sorted by distance ascending.`,
		Example:     "  find-me-gluten-free-pp-cli places near --lat 38.9072 --lng -77.0369 --radius-km 2 --dedicated --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("lat") || !cmd.Flags().Changed("lng") {
				return usageErr(errors.New("--lat and --lng are required"))
			}
			if dryRunOK(flags) {
				return nil
			}
			s, err := openStore(flags)
			if err != nil {
				return err
			}
			defer s.Close()
			rows, err := s.List(resTypeBiz, 0)
			if err != nil {
				return err
			}
			out := []nearResult{}
			for _, raw := range rows {
				var r jsonld.Restaurant
				if json.Unmarshal(raw, &r) != nil {
					continue
				}
				if r.Latitude == 0 && r.Longitude == 0 {
					continue
				}
				d := haversineKm(lat, lng, r.Latitude, r.Longitude)
				if d > radius {
					continue
				}
				if dedicated && !r.Dedicated {
					continue
				}
				if hasGFMenu && !r.HasGFMenu {
					continue
				}
				if minRating > 0 && r.RatingValue < minRating {
					continue
				}
				out = append(out, nearResult{
					BizID:      r.BizID,
					Name:       r.Name,
					Slug:       r.Slug,
					City:       r.City,
					State:      r.State,
					Latitude:   r.Latitude,
					Longitude:  r.Longitude,
					DistanceKm: roundKm(d),
					Rating:     r.RatingValue,
					Dedicated:  r.Dedicated,
					HasGFMenu:  r.HasGFMenu,
				})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].DistanceKm < out[j].DistanceKm })
			if limit > 0 && len(out) > limit {
				out = out[:limit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude of the seed point")
	cmd.Flags().Float64Var(&lng, "lng", 0, "Longitude of the seed point")
	cmd.Flags().Float64Var(&radius, "radius-km", 5, "Radius in kilometers")
	cmd.Flags().BoolVar(&dedicated, "dedicated", false, "Only dedicated gluten-free places")
	cmd.Flags().BoolVar(&hasGFMenu, "has-gf-menu", false, "Only places with a gluten-free menu")
	cmd.Flags().Float64Var(&minRating, "min-rating", 0, "Minimum aggregate rating (0-5)")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum results")
	return cmd
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	rlat1 := lat1 * math.Pi / 180
	rlat2 := lat2 * math.Pi / 180
	dlat := (lat2 - lat1) * math.Pi / 180
	dlon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(rlat1)*math.Cos(rlat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func roundKm(d float64) float64 { return math.Round(d*100) / 100 }

// ============================================================
// places stats --city <c> [--cuisine] [--dedicated]
// ============================================================

type statsResult struct {
	City         string  `json:"city"`
	Cuisine      string  `json:"cuisine,omitempty"`
	Country      string  `json:"country,omitempty"`
	State        string  `json:"state,omitempty"`
	Total        int     `json:"total"`
	Dedicated    int     `json:"dedicated_count"`
	GFMenu       int     `json:"gf_menu_count"`
	DedicatedPct float64 `json:"dedicated_pct"`
	GFMenuPct    float64 `json:"gf_menu_pct"`
	AvgRating    float64 `json:"avg_rating,omitempty"`
	P50Rating    float64 `json:"p50_rating,omitempty"`
	P90Rating    float64 `json:"p90_rating,omitempty"`
	TotalReviews int     `json:"total_reviews"`
	HighestRated string  `json:"highest_rated,omitempty"`
}

func newPlacesStatsCmd(flags *rootFlags) *cobra.Command {
	var cityFlag, cuisineFlag string
	var dedicated bool
	cmd := &cobra.Command{
		Use:         "stats",
		Short:       "Aggregate stats for a city: count, avg rating, p50/p90, dedicated %, GF-menu %",
		Example:     "  find-me-gluten-free-pp-cli places stats --city richmond --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cityFlag == "" {
				return usageErr(errors.New("--city is required"))
			}
			if dryRunOK(flags) {
				return nil
			}
			bizes, err := loadCityBizes(flags, cityFlag)
			if err != nil {
				return err
			}
			if cuisineFlag != "" {
				bizes = filterByCuisineHint(bizes, cuisineFlag)
			}
			result := computeCityStats(cityFlag, bizes, dedicated)
			result.Cuisine = cuisineFlag
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&cityFlag, "city", "", "City slug (e.g., richmond)")
	cmd.Flags().StringVar(&cuisineFlag, "cuisine", "", "Restrict aggregate to bizes whose name or OG description mentions a cuisine (best-effort substring match)")
	cmd.Flags().BoolVar(&dedicated, "dedicated", false, "Restrict aggregate to dedicated-GF places only")
	return cmd
}

// filterByCuisineHint narrows the cached set to bizes whose name or
// og-description mentions the cuisine slug. The site's cuisine taxonomy
// lives in filter sub-pages we don't always sync ahead of time, so the
// cheap fallback is a substring scan of the data we do have.
func filterByCuisineHint(bizes []jsonld.Restaurant, cuisine string) []jsonld.Restaurant {
	target := strings.ToLower(strings.ReplaceAll(cuisine, "-", " "))
	out := make([]jsonld.Restaurant, 0, len(bizes))
	for _, b := range bizes {
		hay := strings.ToLower(b.Name + " " + b.OGDesc)
		if strings.Contains(hay, target) {
			out = append(out, b)
		}
	}
	return out
}

func computeCityStats(city string, bizes []jsonld.Restaurant, dedicatedOnly bool) statsResult {
	r := statsResult{City: city}
	var ratings []float64
	var topName string
	var topRating float64
	for _, b := range bizes {
		if dedicatedOnly && !b.Dedicated {
			continue
		}
		r.Total++
		if b.Dedicated {
			r.Dedicated++
		}
		if b.HasGFMenu {
			r.GFMenu++
		}
		if b.RatingValue > 0 {
			ratings = append(ratings, b.RatingValue)
			if b.RatingValue > topRating || (b.RatingValue == topRating && b.RatingCount > 0) {
				topRating = b.RatingValue
				topName = b.Name
			}
		}
		r.TotalReviews += b.RatingCount
		if b.Country != "" {
			r.Country = b.Country
		}
		if b.State != "" {
			r.State = b.State
		}
	}
	if r.Total > 0 {
		r.DedicatedPct = math.Round(float64(r.Dedicated)/float64(r.Total)*1000) / 10
		r.GFMenuPct = math.Round(float64(r.GFMenu)/float64(r.Total)*1000) / 10
	}
	if len(ratings) > 0 {
		sort.Float64s(ratings)
		var sum float64
		for _, x := range ratings {
			sum += x
		}
		r.AvgRating = math.Round(sum/float64(len(ratings))*100) / 100
		r.P50Rating = ratings[len(ratings)/2]
		r.P90Rating = ratings[int(math.Floor(float64(len(ratings)-1)*0.9))]
		r.HighestRated = topName
	}
	return r
}

// loadCityBizes returns hydrated bizes whose City matches the given slug
// (case-insensitive substring match against the og-derived city name).
func loadCityBizes(flags *rootFlags, citySlug string) ([]jsonld.Restaurant, error) {
	s, err := openStore(flags)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	rows, err := s.List(resTypeBiz, 0)
	if err != nil {
		return nil, err
	}
	target := strings.ToLower(strings.ReplaceAll(citySlug, "-", " "))
	var out []jsonld.Restaurant
	for _, raw := range rows {
		var r jsonld.Restaurant
		if json.Unmarshal(raw, &r) != nil {
			continue
		}
		if strings.ToLower(r.City) == target ||
			strings.ToLower(strings.ReplaceAll(r.City, " ", "-")) == strings.ToLower(citySlug) {
			out = append(out, r)
		}
	}
	return out, nil
}

// ============================================================
// places postal <zip> — postal code resolution
// ============================================================

func newPlacesPostalCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "postal [zip]",
		Short:       "Resolve a postal/zip code to its Find Me Gluten Free city page (follows the redirect)",
		Example:     "  find-me-gluten-free-pp-cli places postal 23219 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			zip := args[0]
			if !validZip(zip) {
				return usageErr(fmt.Errorf("postal code %q is not a recognized format (US ZIP, UK postcode, or country/postal code with letters and digits)", zip))
			}
			if dryRunOK(flags) {
				fmt.Fprintf(cmd.OutOrStdout(), "would resolve /postal/%s\n", zip)
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			req, err := http.NewRequestWithContext(cmd.Context(), "GET", c.BaseURL+"/postal/"+zip, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", fmgfUserAgent)
			noredirect := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
				Timeout:       20 * time.Second,
			}
			resp, err := noredirect.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			loc := resp.Header.Get("Location")
			if loc == "" || loc == "/" {
				return notFoundErr(fmt.Errorf("postal %s does not resolve to a Find Me Gluten Free city page (status %d, redirect: %q)", zip, resp.StatusCode, loc))
			}
			out := map[string]any{
				"postal":      zip,
				"redirect_to": loc,
				"status":      resp.StatusCode,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

// validZip accepts only postal codes with the right shape — digits, optional
// letters, dashes, spaces — between 3 and 12 characters. Anything else is
// pre-rejected before the upstream lookup.
func validZip(zip string) bool {
	if len(zip) < 3 || len(zip) > 12 {
		return false
	}
	for _, r := range zip {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r == '-', r == ' ':
		default:
			return false
		}
	}
	return true
}

// validBizID accepts only Find Me Gluten Free numeric biz IDs (12+ digits).
func validBizID(s string) bool {
	if len(s) < 10 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// validCitySlug accepts URL-safe city slugs (lowercase letters, digits, hyphens).
func validCitySlug(s string) bool {
	if len(s) < 2 || len(s) > 80 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

// ============================================================
// cities sync <country> <state> <city> — fetch city + every biz on the page
// ============================================================

func newCitiesSyncCmd(flags *rootFlags) *cobra.Command {
	var maxBiz int
	cmd := &cobra.Command{
		Use:   "sync [country] [state] [city]",
		Short: "Sync a city: list places, hydrate each one's JSON-LD, take a snapshot for diff",
		Long: `Fetches /{country}/{state}/{city}, extracts every /biz/* link, then
hydrates each business individually (rate-limited). After hydration, writes
a city snapshot row that 'cities diff' uses as a baseline.

This is the foundation command — every other transcendence feature reads
the rows it populates.`,
		Example:     "  find-me-gluten-free-pp-cli cities sync us va richmond",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 3 {
				return usageErr(errors.New("usage: cities sync <country> <state> <city>"))
			}
			country, state, city := args[0], args[1], args[2]
			if dryRunOK(flags) {
				fmt.Fprintf(cmd.OutOrStdout(), "would sync /%s/%s/%s\n", country, state, city)
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			cityPath := fmt.Sprintf("/%s/%s/%s", country, state, city)
			cityHTML, err := getHTML(cmd.Context(), c, cityPath)
			if err != nil {
				return err
			}
			bizPaths := extractBizLinks(cityHTML)
			if maxBiz > 0 && len(bizPaths) > maxBiz {
				bizPaths = bizPaths[:maxBiz]
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "found %d businesses; hydrating with 1 req/sec...\n", len(bizPaths))
			var bizIDs []string
			var failures []string
			for i, p := range bizPaths {
				slug, bizID, ok := splitBizPath(p)
				if !ok {
					continue
				}
				if i > 0 {
					time.Sleep(1 * time.Second)
				}
				r, err := hydrateOne(cmd.Context(), c, slug, bizID)
				if err != nil {
					failures = append(failures, fmt.Sprintf("%s: %v", p, err))
					continue
				}
				if err := persistBiz(flags, r); err != nil {
					failures = append(failures, fmt.Sprintf("%s: persist: %v", p, err))
					continue
				}
				bizIDs = append(bizIDs, bizID)
				fmt.Fprintf(cmd.ErrOrStderr(), "  [%d/%d] %s\n", i+1, len(bizPaths), r.Name)
			}
			snap := citySnapshot{
				CountryState: fmt.Sprintf("%s/%s", country, state),
				City:         city,
				BizIDs:       bizIDs,
				TakenAt:      time.Now().UTC(),
				Failures:     failures,
			}
			if err := persistSnapshot(flags, snap); err != nil {
				return err
			}
			out := map[string]any{
				"city":     city,
				"country":  country,
				"state":    state,
				"hydrated": len(bizIDs),
				"failures": len(failures),
				"snapshot": snap.id(),
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().IntVar(&maxBiz, "limit", 0, "Hydrate at most N businesses (0 = no limit)")
	return cmd
}

func extractBizLinks(htmlBody []byte) []string {
	body := string(htmlBody)
	seen := map[string]struct{}{}
	var out []string
	for _, m := range bizLinkRegex.FindAllStringSubmatch(body, -1) {
		path := "/biz/" + m[1] + "/" + m[2]
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

// splitBizPath parses "/biz/<slug>/<id>" or just "biz/<slug>/<id>" forms.
var bizPathRegex = regexp.MustCompile(`/?biz/([^/]+)/(\d+)`)

func splitBizPath(p string) (slug, bizID string, ok bool) {
	m := bizPathRegex.FindStringSubmatch(p)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// ============================================================
// cities diff <city> --since <ts|last-sync>
// ============================================================

type citySnapshot struct {
	CountryState string    `json:"country_state"`
	City         string    `json:"city"`
	BizIDs       []string  `json:"biz_ids"`
	TakenAt      time.Time `json:"taken_at"`
	Failures     []string  `json:"failures,omitempty"`
}

func (s citySnapshot) id() string {
	return fmt.Sprintf("%s::%s::%s", s.CountryState, s.City, s.TakenAt.Format("20060102-150405"))
}

func persistSnapshot(flags *rootFlags, s citySnapshot) error {
	st, err := openStore(flags)
	if err != nil {
		return err
	}
	defer st.Close()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return st.Upsert(resTypeSnapshot, storeID(resTypeSnapshot, s.id()), data)
}

func loadSnapshots(flags *rootFlags, city string) ([]citySnapshot, error) {
	st, err := openStore(flags)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	rows, err := st.List(resTypeSnapshot, 0)
	if err != nil {
		return nil, err
	}
	var out []citySnapshot
	target := strings.ToLower(city)
	for _, raw := range rows {
		var s citySnapshot
		if json.Unmarshal(raw, &s) != nil {
			continue
		}
		if strings.ToLower(s.City) == target {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TakenAt.Before(out[j].TakenAt) })
	return out, nil
}

type cityDiff struct {
	City         string   `json:"city"`
	From         string   `json:"from"`
	To           string   `json:"to"`
	Added        []string `json:"added"`
	Removed      []string `json:"removed"`
	BizCountFrom int      `json:"biz_count_from"`
	BizCountTo   int      `json:"biz_count_to"`
	Delta        int      `json:"delta"`
}

func newCitiesDiffCmd(flags *rootFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:   "diff [city]",
		Short: "Show what changed in a city since the last sync (added/removed places, count delta)",
		Long: `Compares the two most recent city snapshots written by 'cities sync'. To get
useful output, run 'cities sync <country> <state> <city>' twice with time
between (or after the upstream changes). The first sync establishes a
baseline; the second sync writes a new snapshot that this command diffs
against the previous.

--since accepts the literal string 'last-sync' (default; compares the two
most recent snapshots) or a YYYY-MM-DD date (uses the most recent snapshot
on or before that date as the baseline).`,
		Example:     "  find-me-gluten-free-pp-cli cities diff richmond --since last-sync --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			city := args[0]
			if dryRunOK(flags) {
				return nil
			}
			snaps, err := loadSnapshots(flags, city)
			if err != nil {
				return err
			}
			if len(snaps) < 2 {
				return notFoundErr(fmt.Errorf("need 2+ snapshots for %s; have %d. Run 'cities sync' to add more", city, len(snaps)))
			}
			cur := snaps[len(snaps)-1]
			prev := snaps[len(snaps)-2]
			if since != "" && since != "last-sync" {
				cutoff, err := time.Parse("2006-01-02", since)
				if err != nil {
					return usageErr(fmt.Errorf("--since must be 'last-sync' or YYYY-MM-DD: %w", err))
				}
				for i := len(snaps) - 1; i >= 0; i-- {
					if !snaps[i].TakenAt.After(cutoff) {
						prev = snaps[i]
						break
					}
				}
			}
			added, removed := diffIDLists(prev.BizIDs, cur.BizIDs)
			out := cityDiff{
				City:         city,
				From:         prev.TakenAt.Format(time.RFC3339),
				To:           cur.TakenAt.Format(time.RFC3339),
				Added:        added,
				Removed:      removed,
				BizCountFrom: len(prev.BizIDs),
				BizCountTo:   len(cur.BizIDs),
				Delta:        len(cur.BizIDs) - len(prev.BizIDs),
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&since, "since", "last-sync", "Baseline: 'last-sync' or YYYY-MM-DD")
	return cmd
}

func diffIDLists(prev, cur []string) (added, removed []string) {
	pm := map[string]struct{}{}
	for _, id := range prev {
		pm[id] = struct{}{}
	}
	cm := map[string]struct{}{}
	for _, id := range cur {
		cm[id] = struct{}{}
	}
	for id := range cm {
		if _, ok := pm[id]; !ok {
			added = append(added, id)
		}
	}
	for id := range pm {
		if _, ok := cm[id]; !ok {
			removed = append(removed, id)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// ============================================================
// trip compare <city1> <city2> [<cityN>]
// ============================================================

func newTripCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "trip", Short: "Trip planning utilities (compare, plan)"}
	cmd.AddCommand(newTripCompareCmd(flags))
	return cmd
}

func newTripCompareCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare [cities...]",
		Short: "Compare 2+ cities side-by-side: dedicated count, GF-menu count, average rating, top-rated picks",
		Long: `Reads the locally hydrated business cache for each named city slug (run
'cities sync' first for each) and emits a per-city aggregate. Useful for
choosing between trip stops by celiac-friendliness.`,
		Example:     "  find-me-gluten-free-pp-cli trip compare lisbon porto madrid --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return usageErr(errors.New("trip compare requires 2+ city slugs"))
			}
			if dryRunOK(flags) {
				return nil
			}
			results := make([]statsResult, 0, len(args))
			for _, city := range args {
				bizes, err := loadCityBizes(flags, city)
				if err != nil {
					return err
				}
				results = append(results, computeCityStats(city, bizes, false))
			}
			return printJSONFiltered(cmd.OutOrStdout(), results, flags)
		},
	}
	return cmd
}

// ============================================================
// reviews recent / reviews safety
// ============================================================

func newReviewsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reviews",
		Short: "City-wide review feed and per-place safety filters",
	}
	cmd.AddCommand(newReviewsRecentCmd(flags))
	cmd.AddCommand(newReviewsSafetyCmd(flags))
	return cmd
}

type reviewRow struct {
	BizID         string  `json:"biz_id"`
	BizName       string  `json:"biz_name"`
	BizSlug       string  `json:"biz_slug,omitempty"`
	City          string  `json:"city,omitempty"`
	Rating        float64 `json:"rating,omitempty"`
	DatePublished string  `json:"date_published,omitempty"`
	Author        string  `json:"author,omitempty"`
	Description   string  `json:"description,omitempty"`
}

func newReviewsRecentCmd(flags *rootFlags) *cobra.Command {
	var cityFlag, sinceFlag string
	var maxRating, minRating float64
	var limit int
	cmd := &cobra.Command{
		Use:         "recent",
		Short:       "City-wide newest reviews. Use --max-rating 2 to surface safety-report-style 1-2 star reviews.",
		Example:     "  find-me-gluten-free-pp-cli reviews recent --city portland --max-rating 2 --since 2026-04-01 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cityFlag == "" {
				return usageErr(errors.New("--city is required"))
			}
			if dryRunOK(flags) {
				return nil
			}
			bizes, err := loadCityBizes(flags, cityFlag)
			if err != nil {
				return err
			}
			var since time.Time
			if sinceFlag != "" {
				t, err := time.Parse("2006-01-02", sinceFlag)
				if err != nil {
					return usageErr(fmt.Errorf("--since must be YYYY-MM-DD: %w", err))
				}
				since = t
			}
			out := []reviewRow{}
			for _, b := range bizes {
				for _, rv := range b.Reviews {
					if maxRating > 0 && rv.Rating > maxRating {
						continue
					}
					if minRating > 0 && rv.Rating < minRating {
						continue
					}
					if !since.IsZero() {
						t, err := time.Parse("2006-01-02", rv.DatePublished)
						if err != nil || t.Before(since) {
							continue
						}
					}
					out = append(out, reviewRow{
						BizID:         b.BizID,
						BizName:       b.Name,
						BizSlug:       b.Slug,
						City:          b.City,
						Rating:        rv.Rating,
						DatePublished: rv.DatePublished,
						Author:        rv.Author,
						Description:   rv.Description,
					})
				}
			}
			sort.Slice(out, func(i, j int) bool { return out[i].DatePublished > out[j].DatePublished })
			if limit > 0 && len(out) > limit {
				out = out[:limit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&cityFlag, "city", "", "City slug (required)")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Earliest review date (YYYY-MM-DD)")
	cmd.Flags().Float64Var(&maxRating, "max-rating", 0, "Surface only reviews with rating <= N (use 2 for safety reports)")
	cmd.Flags().Float64Var(&minRating, "min-rating", 0, "Surface only reviews with rating >= N")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum reviews to return")
	return cmd
}

func newReviewsSafetyCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "safety [biz_id]",
		Short:       "Filter cached reviews for a place to those mentioning got-glutened / cross-contamination keywords (deterministic, no LLM)",
		Example:     "  find-me-gluten-free-pp-cli reviews safety 6118524171452416 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			b, err := loadBiz(flags, args[0])
			if err != nil {
				return notFoundErr(fmt.Errorf("biz %s not in local cache; run: places hydrate <slug> %s", args[0], args[0]))
			}
			out := []reviewRow{}
			for _, rv := range b.Reviews {
				if !safety.IsSafetyReport(rv.Description) {
					continue
				}
				out = append(out, reviewRow{
					BizID:         b.BizID,
					BizName:       b.Name,
					Rating:        rv.Rating,
					DatePublished: rv.DatePublished,
					Author:        rv.Author,
					Description:   rv.Description,
				})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].DatePublished > out[j].DatePublished })
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

// ============================================================
// watch add / list / report
// ============================================================

type watchEntry struct {
	BizID               string    `json:"biz_id"`
	BizName             string    `json:"biz_name,omitempty"`
	AddedAt             time.Time `json:"added_at"`
	BaselineRating      float64   `json:"baseline_rating,omitempty"`
	BaselineReviewCount int       `json:"baseline_review_count,omitempty"`
}

func newWatchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Track specific places for rating drift and new low-rated reviews",
	}
	cmd.AddCommand(newWatchAddCmd(flags))
	cmd.AddCommand(newWatchListCmd(flags))
	cmd.AddCommand(newWatchRemoveCmd(flags))
	cmd.AddCommand(newWatchReportCmd(flags))
	return cmd
}

func newWatchAddCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "add [biz_id]",
		Short:       "Add a place to your watchlist; record baseline rating + review count",
		Example:     "  find-me-gluten-free-pp-cli watch add 6118524171452416 --json",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if !validBizID(args[0]) {
				return usageErr(fmt.Errorf("biz_id %q is not a valid Find Me Gluten Free numeric ID (10+ digits)", args[0]))
			}
			if dryRunOK(flags) {
				return nil
			}
			bizID := args[0]
			b, err := loadBiz(flags, bizID)
			if err != nil {
				return notFoundErr(fmt.Errorf("biz %s not in cache; run 'places hydrate' first", bizID))
			}
			entry := watchEntry{
				BizID:               bizID,
				BizName:             b.Name,
				AddedAt:             time.Now().UTC(),
				BaselineRating:      b.RatingValue,
				BaselineReviewCount: b.RatingCount,
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			data, _ := json.Marshal(entry)
			if err := st.Upsert(resTypeWatch, storeID(resTypeWatch, bizID), data); err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), entry, flags)
		},
	}
	return cmd
}

func newWatchListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List every place on your watchlist",
		Example:     "  find-me-gluten-free-pp-cli watch list --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			entries, err := loadWatchlist(flags)
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), entries, flags)
		},
	}
	return cmd
}

func newWatchRemoveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "remove [biz_id]",
		Short:       "Remove a place from your watchlist",
		Example:     "  find-me-gluten-free-pp-cli watch remove 6118524171452416",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.DB().Exec(`DELETE FROM resources WHERE resource_type = ? AND id = ?`, resTypeWatch, storeID(resTypeWatch, args[0]))
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				return notFoundErr(fmt.Errorf("no watch entry found for biz_id %q", args[0]))
			}
			return nil
		},
	}
	return cmd
}

type watchReportRow struct {
	BizID            string      `json:"biz_id"`
	BizName          string      `json:"biz_name,omitempty"`
	BaselineRating   float64     `json:"baseline_rating"`
	CurrentRating    float64     `json:"current_rating"`
	RatingDrift      float64     `json:"rating_drift"`
	NewReviews       int         `json:"new_reviews"`
	LowRatingReviews []reviewRow `json:"low_rating_reviews,omitempty"`
}

func newWatchReportCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "report",
		Short:       "Re-fetch every watched place and report rating drift + new low-rated reviews",
		Example:     "  find-me-gluten-free-pp-cli watch report --json",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			entries, err := loadWatchlist(flags)
			if err != nil {
				return err
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			out := []watchReportRow{}
			for i, e := range entries {
				if i > 0 {
					time.Sleep(1 * time.Second)
				}
				b, err := hydrateOne(cmd.Context(), c, slugFromName(e.BizName, e.BizID), e.BizID)
				if err != nil {
					continue
				}
				if err := persistBiz(flags, b); err != nil {
					return err
				}
				row := watchReportRow{
					BizID:          b.BizID,
					BizName:        b.Name,
					BaselineRating: e.BaselineRating,
					CurrentRating:  b.RatingValue,
					RatingDrift:    math.Round((b.RatingValue-e.BaselineRating)*100) / 100,
					NewReviews:     b.RatingCount - e.BaselineReviewCount,
				}
				for _, rv := range b.Reviews {
					if rv.Rating <= 2 {
						row.LowRatingReviews = append(row.LowRatingReviews, reviewRow{
							BizID:         b.BizID,
							BizName:       b.Name,
							Rating:        rv.Rating,
							DatePublished: rv.DatePublished,
							Author:        rv.Author,
							Description:   rv.Description,
						})
					}
				}
				out = append(out, row)
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

func slugFromName(name, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "'", "")
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '/':
			return '-'
		default:
			return -1
		}
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if s == "" {
		s = fallback
	}
	return s
}

func loadWatchlist(flags *rootFlags) ([]watchEntry, error) {
	st, err := openStore(flags)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	rows, err := st.List(resTypeWatch, 0)
	if err != nil {
		return nil, err
	}
	out := []watchEntry{}
	for _, raw := range rows {
		var e watchEntry
		if json.Unmarshal(raw, &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

// ============================================================
// cuisines compare <cuisine> <city1> <city2>...
// ============================================================

type cuisineCityResult struct {
	City      string  `json:"city"`
	Cuisine   string  `json:"cuisine"`
	BizCount  int     `json:"biz_count"`
	TopRated  string  `json:"top_rated,omitempty"`
	TopRating float64 `json:"top_rating,omitempty"`
}

func newCuisinesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cuisines",
		Short: "Cuisine-aware queries (compare across cities)",
	}
	cmd.AddCommand(newCuisinesCompareCmd(flags))
	return cmd
}

func newCuisinesCompareCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare [cuisine] [country] [state] [cities...]",
		Short: "Compare a cuisine across multiple cities (counts, top-rated picks)",
		Long: `Fetches /{country}/{state}/{city}/{cuisine} for each named city and
joins against the local hydrated business cache to surface counts and the
top-rated place per city for that cuisine.`,
		Example:     "  find-me-gluten-free-pp-cli cuisines compare burgers us va richmond charlottesville --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 4 {
				return usageErr(errors.New("usage: cuisines compare <cuisine> <country> <state> <city...> (2+ cities)"))
			}
			cuisine := args[0]
			country, state := args[1], args[2]
			cities := args[3:]
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			out := []cuisineCityResult{}
			for i, city := range cities {
				if i > 0 {
					time.Sleep(1 * time.Second)
				}
				cityPath := fmt.Sprintf("/%s/%s/%s/%s", country, state, city, cuisine)
				body, err := getHTML(cmd.Context(), c, cityPath)
				if err != nil {
					out = append(out, cuisineCityResult{City: city, Cuisine: cuisine})
					continue
				}
				bizPaths := extractBizLinks(body)
				row := cuisineCityResult{City: city, Cuisine: cuisine, BizCount: len(bizPaths)}
				// Look up top-rated from local cache among these biz IDs
				for _, p := range bizPaths {
					_, bizID, ok := splitBizPath(p)
					if !ok {
						continue
					}
					b, err := loadBiz(flags, bizID)
					if err != nil || b == nil {
						continue
					}
					if b.RatingValue > row.TopRating {
						row.TopRating = b.RatingValue
						row.TopRated = b.Name
					}
				}
				out = append(out, row)
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

// ============================================================
// bookmark add / list / remove
// ============================================================

type bookmarkEntry struct {
	BizID   string    `json:"biz_id"`
	BizName string    `json:"biz_name,omitempty"`
	City    string    `json:"city,omitempty"`
	AddedAt time.Time `json:"added_at"`
	Note    string    `json:"note,omitempty"`
}

func newBookmarkCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookmark",
		Short: "Save and manage your favorite restaurants (local-only, no premium subscription)",
	}
	cmd.AddCommand(newBookmarkAddCmd(flags))
	cmd.AddCommand(newBookmarkListCmd(flags))
	cmd.AddCommand(newBookmarkRemoveCmd(flags))
	return cmd
}

func newBookmarkAddCmd(flags *rootFlags) *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:         "add [biz_id]",
		Short:       "Bookmark a hydrated place",
		Example:     "  find-me-gluten-free-pp-cli bookmark add 6118524171452416 --note \"Anniversary spot\" --json",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if !validBizID(args[0]) {
				return usageErr(fmt.Errorf("biz_id %q is not a valid Find Me Gluten Free numeric ID (10+ digits)", args[0]))
			}
			if dryRunOK(flags) {
				return nil
			}
			bizID := args[0]
			b, _ := loadBiz(flags, bizID)
			entry := bookmarkEntry{BizID: bizID, AddedAt: time.Now().UTC(), Note: note}
			if b != nil {
				entry.BizName = b.Name
				entry.City = b.City
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			data, _ := json.Marshal(entry)
			if err := st.Upsert(resTypeBookmark, storeID(resTypeBookmark, bizID), data); err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), entry, flags)
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "Optional personal note")
	return cmd
}

func newBookmarkListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List every bookmarked place",
		Example:     "  find-me-gluten-free-pp-cli bookmark list --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			rows, err := st.List(resTypeBookmark, 0)
			if err != nil {
				return err
			}
			out := []bookmarkEntry{}
			for _, raw := range rows {
				var e bookmarkEntry
				if json.Unmarshal(raw, &e) == nil {
					out = append(out, e)
				}
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

func newBookmarkRemoveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "remove [biz_id]",
		Short:       "Remove a bookmarked place by biz_id from the local bookmark store",
		Example:     "  find-me-gluten-free-pp-cli bookmark remove 6118524171452416",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.DB().Exec(`DELETE FROM resources WHERE resource_type = ? AND id = ?`, resTypeBookmark, storeID(resTypeBookmark, args[0]))
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				return notFoundErr(fmt.Errorf("no bookmark found for biz_id %q", args[0]))
			}
			return nil
		},
	}
	return cmd
}

// ============================================================
// cities save / unsave / saved
// ============================================================

type savedCityEntry struct {
	CitySlug string    `json:"city_slug"`
	AddedAt  time.Time `json:"added_at"`
	Note     string    `json:"note,omitempty"`
}

func newCitiesSaveCmd(flags *rootFlags) *cobra.Command {
	var note string
	cmd := &cobra.Command{
		Use:         "save [city]",
		Short:       "Save a city slug for quick recall (local-only)",
		Example:     "  find-me-gluten-free-pp-cli cities save richmond --note \"family trip 2026\" --json",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if !validCitySlug(args[0]) {
				return usageErr(fmt.Errorf("city slug %q is not in the expected lowercase-kebab form (e.g., richmond, charlottesville-virginia)", args[0]))
			}
			if dryRunOK(flags) {
				return nil
			}
			entry := savedCityEntry{CitySlug: args[0], AddedAt: time.Now().UTC(), Note: note}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			data, _ := json.Marshal(entry)
			if err := st.Upsert(resTypeSavedCity, storeID(resTypeSavedCity, entry.CitySlug), data); err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), entry, flags)
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "Optional note")
	return cmd
}

func newCitiesUnsaveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "unsave [city]",
		Short:       "Remove a saved city",
		Example:     "  find-me-gluten-free-pp-cli cities unsave richmond",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.DB().Exec(`DELETE FROM resources WHERE resource_type = ? AND id = ?`, resTypeSavedCity, storeID(resTypeSavedCity, args[0]))
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				return notFoundErr(fmt.Errorf("no saved city found for slug %q", args[0]))
			}
			return nil
		},
	}
	return cmd
}

func newCitiesSavedCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "saved",
		Short:       "List cities saved locally for quick recall, with their slug, save time, and optional notes",
		Example:     "  find-me-gluten-free-pp-cli cities saved --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			st, err := openStore(flags)
			if err != nil {
				return err
			}
			defer st.Close()
			rows, err := st.List(resTypeSavedCity, 0)
			if err != nil {
				return err
			}
			out := []savedCityEntry{}
			for _, raw := range rows {
				var e savedCityEntry
				if json.Unmarshal(raw, &e) == nil {
					out = append(out, e)
				}
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}
