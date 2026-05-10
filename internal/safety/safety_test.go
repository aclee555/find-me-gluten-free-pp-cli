package safety

import "testing"

func TestIsSafetyReport(t *testing.T) {
	cases := []struct {
		name string
		desc string
		want bool
	}{
		{"plain positive review", "Loved the bread, fluffy and great!", false},
		{"got glutened explicit", "I got glutened here last week.", true},
		{"glutened lowercase", "Definitely glutened me.", true},
		{"cross-contam hyphen", "Saw flour on every surface — clearly cross-contamination.", true},
		{"shared fryer", "They use a shared fryer so my fries had cross contam.", true},
		{"reacted symptom", "Felt sick within an hour, definitely reacted.", true},
		{"empty", "", false},
		{"happy review with the word fryer", "Has a dedicated fryer which is great.", false},
		{"contaminated keyword", "I think the food was contaminated with gluten.", true},
		{"flour everywhere", "Flour everywhere in the open kitchen.", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSafetyReport(tc.desc); got != tc.want {
				t.Errorf("IsSafetyReport(%q) = %v, want %v", tc.desc, got, tc.want)
			}
		})
	}
}
