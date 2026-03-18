package main

import "testing"

func TestNormalizeVisibility(t *testing.T) {
	cases := []struct {
		in      string
		out     string
		wantErr bool
	}{
		{in: "", out: "公开可见"},
		{in: "public", out: "公开可见"},
		{in: "PUBLIC", out: "公开可见"},
		{in: "self-only", out: "仅自己可见"},
		{in: "private", out: "仅自己可见"},
		{in: "friends-only", out: "仅互关好友可见"},
		{in: "mutual_follow", out: "仅互关好友可见"},
		{in: "公开可见", out: "公开可见"},
		{in: "仅自己可见", out: "仅自己可见"},
		{in: "仅互关好友可见", out: "仅互关好友可见"},
		{in: "unknown", wantErr: true},
	}

	for _, tc := range cases {
		got, err := normalizeVisibility(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeVisibility(%q) expected error, got none", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeVisibility(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.out {
			t.Fatalf("normalizeVisibility(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}
