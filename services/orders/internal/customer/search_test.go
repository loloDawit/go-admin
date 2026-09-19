package customer

import "testing"

// ILIKE treats % and _ as wildcards, so a search for "50%" would otherwise
// match everything beginning "50". The term is data, not a pattern.
func TestSearchParamEscapesWildcards(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"50%", `%50\%%`},
		{"a_b", `%a\_b%`},
		{`back\slash`, `%back\\slash%`},
		{"plain", "%plain%"},
	} {
		got := searchParam(tc.in)
		if got == nil {
			t.Fatalf("searchParam(%q) = nil, want a pattern", tc.in)
		}
		if *got != tc.want {
			t.Errorf("searchParam(%q) = %q, want %q", tc.in, *got, tc.want)
		}
	}

	if searchParam("") != nil {
		t.Error("an empty term must stay nil, which the filter clause reads as no filter")
	}
}
