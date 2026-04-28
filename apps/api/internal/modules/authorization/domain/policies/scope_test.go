package policies

import "testing"

func TestMatchesScope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                                   string
		grantedType, grantedID, reqType, reqID string
		want                                   bool
	}{
		{"empty req always matches", "tower", "T1", "", "", true},
		{"tenant grant covers tower", "tenant", "", "tower", "T1", true},
		{"tenant grant covers unit", "tenant", "", "unit", "U-101", true},
		{"empty grant covers everything", "", "", "tower", "T1", true},
		{"tower grant matches same tower", "tower", "T1", "tower", "T1", true},
		{"tower grant rejects different tower", "tower", "T1", "tower", "T2", false},
		{"unit grant matches same unit", "unit", "U-101", "unit", "U-101", true},
		{"unit grant rejects different unit", "unit", "U-101", "unit", "U-202", false},
		{"tower grant cannot escalate to tenant", "tower", "T1", "tenant", "", false},
		{"unit grant cannot answer tower scope", "unit", "U-101", "tower", "T1", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := MatchesScope(tc.grantedType, tc.grantedID, tc.reqType, tc.reqID)
			if got != tc.want {
				t.Fatalf("MatchesScope(%q,%q,%q,%q) = %v; want %v",
					tc.grantedType, tc.grantedID, tc.reqType, tc.reqID, got, tc.want)
			}
		})
	}
}
