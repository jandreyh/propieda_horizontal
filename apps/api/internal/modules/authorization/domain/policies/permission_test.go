package policies

import "testing"

func TestHasPermission(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		granted  []string
		required string
		want     bool
	}{
		{"empty granted denies", []string{}, "package.read", false},
		{"nil granted denies", nil, "package.read", false},
		{"empty required denies", []string{"package.read"}, "", false},
		{"exact match allows", []string{"package.read"}, "package.read", true},
		{"different ns denies", []string{"package.read"}, "visit.read", false},
		{"prefix wildcard allows same ns", []string{"package.*"}, "package.deliver", true},
		{"prefix wildcard does not cross ns", []string{"package.*"}, "visit.deliver", false},
		{"global wildcard allows", []string{"*"}, "anything.allowed", true},
		{"empty entry skipped", []string{"", "package.read"}, "package.read", true},
		{"multiple grants any matches", []string{"visit.read", "package.deliver"}, "package.deliver", true},
		{"wildcard with empty required still denies", []string{"*"}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasPermission(tc.granted, tc.required); got != tc.want {
				t.Fatalf("HasPermission(%v, %q) = %v; want %v", tc.granted, tc.required, got, tc.want)
			}
		})
	}
}

func TestHasAnyPermission(t *testing.T) {
	t.Parallel()

	if !HasAnyPermission([]string{"package.read"}, "visit.read", "package.read") {
		t.Fatal("expected at least one match")
	}
	if HasAnyPermission([]string{"a.b"}, "x.y", "z.w") {
		t.Fatal("expected no match")
	}
	if HasAnyPermission(nil, "any") {
		t.Fatal("expected deny on empty granted")
	}
}
