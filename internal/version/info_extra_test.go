package version

import "testing"

func TestVersionInfoHelpers(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"", "dev"}, {"(devel)", "dev"}, {"dev", "dev"}, {"1.2.3", "v1.2.3"}, {" v1.2.3 ", "v1.2.3"},
	} {
		if got := Normalize(tc.input); got != tc.want {
			t.Fatalf("Normalize(%q) = %q", tc.input, got)
		}
	}
	info := Resolve(Info{Version: "1.2.3", Commit: " abc ", Date: " 2026-01-01 "})
	if info.Version != "v1.2.3" || info.Commit != "abc" || info.Date != "2026-01-01" {
		t.Fatalf("Resolve = %+v", info)
	}
	if !sameVersion("v1.2.3", "1.2.3") || sameVersion("", "v1.2.3") || sameVersion("v1.2.3", "v1.2.4") {
		t.Fatal("sameVersion failed")
	}
	if got := displayCommit("unknown"); got == "" {
		t.Fatal("displayCommit should return a visible fallback")
	}
}
