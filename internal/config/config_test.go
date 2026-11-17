package config

import "testing"

func TestClipNoticeDefaultOn(t *testing.T) {
	var cur File
	if err := decodeFile([]byte(`{"dataDir":"x"}`), &cur); err != nil {
		t.Fatal(err)
	}
	if !cur.ClipNotice {
		t.Fatal("missing clipNotice must default on")
	}
}

func TestClipNoticeOffPersists(t *testing.T) {
	var cur File
	if err := decodeFile([]byte(`{"dataDir":"x","clipNotice":false}`), &cur); err != nil {
		t.Fatal(err)
	}
	if cur.ClipNotice {
		t.Fatal("false must stay off")
	}
}
