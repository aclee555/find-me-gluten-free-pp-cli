package jsonld

import (
	"strings"
	"testing"
)

const samplePinkysHTML = `<html><head>
<meta property="og:title" content="Pinky&#039;s - Gluten-Free Restaurant in Richmond, VA"/>
<meta property="og:description" content="Not dedicated gluten-free but has a gluten-free menu, and is reported to offer gluten-free bread/buns, fries, sandwiches, fried chicken and more."/>
<script type="application/ld+json">
{
  "@context": "http://schema.org",
  "@type": "Restaurant",
  "image":"https://example.com/img.jpg",
  "name": "Pinky&#039;s",
  "priceRange": "$$",
  "telephone": "(804) 802-4716",
  "location": {
    "@type": "Place",
    "geo": {
      "@type": "GeoCoordinates",
      "latitude": "37.568386",
      "longitude": "-77.469025"
    }
  },
  "aggregateRating":{
    "ratingCount":78,
    "@type":"AggregateRating",
    "ratingValue":5.0
  },
  "review":[
    {
      "reviewRating":{"ratingValue":5},
      "datePublished":"2026-04-01",
      "description":"Loved it. The bread is udis &amp; tasted great.",
      "author": {"@type": "Person", "name": "desi_summer"}
    },
    {
      "reviewRating":{"ratingValue":1},
      "datePublished":"2026-03-15",
      "description":"I got glutened &mdash; cross-contamination from the fryer.",
      "author": {"@type": "Person", "name": "celiac_visitor"}
    }
  ]
}
</script>
</head></html>`

func TestParse_HappyPath(t *testing.T) {
	r, err := Parse([]byte(samplePinkysHTML), "https://www.findmeglutenfree.com/biz/pinkys/6118524171452416")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if r.Name != "Pinky's" {
		t.Errorf("Name decoded wrong: %q (want Pinky's)", r.Name)
	}
	if r.Telephone != "(804) 802-4716" {
		t.Errorf("Telephone: %q", r.Telephone)
	}
	if r.Latitude < 37.5 || r.Latitude > 37.6 {
		t.Errorf("Latitude not parsed: %v", r.Latitude)
	}
	if r.Longitude > -77.46 || r.Longitude < -77.48 {
		t.Errorf("Longitude not parsed: %v", r.Longitude)
	}
	if r.RatingValue != 5.0 {
		t.Errorf("RatingValue: %v", r.RatingValue)
	}
	if r.RatingCount != 78 {
		t.Errorf("RatingCount: %v", r.RatingCount)
	}
	if len(r.Reviews) != 2 {
		t.Fatalf("Reviews len: %d", len(r.Reviews))
	}
	if !strings.Contains(r.Reviews[0].Description, "udis & tasted") {
		t.Errorf("HTML entity not decoded in review: %q", r.Reviews[0].Description)
	}
	if !strings.Contains(r.Reviews[1].Description, "got glutened — cross") {
		t.Errorf("mdash entity not decoded: %q", r.Reviews[1].Description)
	}
	if r.City != "Richmond" {
		t.Errorf("City from og:title: %q", r.City)
	}
	if r.State != "VA" {
		t.Errorf("State from og:title: %q", r.State)
	}
	if r.HasGFMenu != true {
		t.Errorf("HasGFMenu should be true (og:description says 'gluten-free menu')")
	}
	if r.Dedicated != false {
		t.Errorf("Dedicated should be false (og:description says 'not dedicated')")
	}
	if r.BizID != "6118524171452416" {
		t.Errorf("BizID: %q", r.BizID)
	}
	if r.Slug != "pinkys" {
		t.Errorf("Slug: %q", r.Slug)
	}
}

func TestParse_DedicatedDetection(t *testing.T) {
	html := `<head>
<meta property="og:description" content="Dedicated gluten-free bakery and cafe."/>
<script type="application/ld+json">{"@context":"http://schema.org","@type":"Restaurant","name":"All GF Bakery"}</script>
</head>`
	r, err := Parse([]byte(html), "https://www.findmeglutenfree.com/biz/all-gf/123")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Dedicated {
		t.Errorf("expected Dedicated=true for 'Dedicated gluten-free bakery' description")
	}
	if !r.HasGFMenu {
		t.Errorf("expected HasGFMenu=true (dedicated implies has-gf-menu)")
	}
}

func TestParse_NoJSONLD(t *testing.T) {
	html := `<html><body>Some page without JSON-LD</body></html>`
	_, err := Parse([]byte(html), "https://www.findmeglutenfree.com/")
	if err == nil {
		t.Errorf("expected error for missing JSON-LD")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	html := `<script type="application/ld+json">{not valid json}</script>`
	_, err := Parse([]byte(html), "https://example.com/biz/x/1")
	if err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

func TestParse_LiteralNewlinesInDescription(t *testing.T) {
	// Reproduces the live Find Me Gluten Free bug: literal \n\n inside
	// review.description string values, which strict JSON forbids.
	html := "<script type=\"application/ld+json\">\n{\n\"@context\":\"http://schema.org\",\n\"@type\":\"Restaurant\",\n\"name\":\"Site\",\n\"review\":[{\"reviewRating\":{\"ratingValue\":5},\"datePublished\":\"2026-01-01\",\"description\":\"Great place.\n\nReally tasty.\"}]\n}\n</script>"
	r, err := Parse([]byte(html), "https://www.findmeglutenfree.com/biz/site/1")
	if err != nil {
		t.Fatalf("Parse should normalize literal newlines: %v", err)
	}
	if len(r.Reviews) != 1 || !strings.Contains(r.Reviews[0].Description, "Really tasty") {
		t.Errorf("expected review with both lines; got %+v", r.Reviews)
	}
}

func TestParse_PartialFields(t *testing.T) {
	// Newly listed places sometimes have name + image but no rating yet.
	html := `<script type="application/ld+json">
{"@context":"http://schema.org","@type":"Restaurant","name":"New Place","image":"http://example.com/img.jpg"}
</script>`
	r, err := Parse([]byte(html), "https://www.findmeglutenfree.com/biz/new-place/9999")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if r.Name != "New Place" {
		t.Errorf("Name: %q", r.Name)
	}
	if r.RatingValue != 0 {
		t.Errorf("RatingValue should be 0 for missing aggregateRating")
	}
	if len(r.Reviews) != 0 {
		t.Errorf("Reviews should be empty")
	}
}
