package winapp

import "testing"

func TestAboutLines(t *testing.T) {
	cases := []struct {
		ver, rev, wantVer, wantRev string
	}{
		{"dev", "4c0e069-dirty", "Development build", "Commit 4c0e069-dirty"},
		{"dev", "", "Development build", ""},
		{"0.0.1", "4c0e069", "Version v0.0.1", "Commit 4c0e069"},
		{"v0.0.1", "abc1234", "Version v0.0.1", "Commit abc1234"},
		{"", "deadbee", "Development build", "Commit deadbee"},
	}
	for _, c := range cases {
		gotVer, gotRev := aboutLines(c.ver, c.rev)
		if gotVer != c.wantVer || gotRev != c.wantRev {
			t.Fatalf("aboutLines(%q, %q) = %q, %q; want %q, %q", c.ver, c.rev, gotVer, gotRev, c.wantVer, c.wantRev)
		}
	}
}
