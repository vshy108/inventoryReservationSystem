package main

import "testing"

func TestParseSeed(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
		wantLen int
	}{
		{"empty", "", false, 0},
		{"whitespace", "   ", false, 0},
		{"single", "sku-1:5", false, 1},
		{"multi", "sku-1:1, sku-2:2 , sku-3:0", false, 3},
		{"missing colon", "sku-1", true, 0},
		{"empty id", ":5", true, 0},
		{"non-int stock", "sku-1:abc", true, 0},
		{"negative stock", "sku-1:-1", true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseSeed(c.in)
			if c.wantErr {
				if err == nil {
					t.Errorf("want error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != c.wantLen {
				t.Errorf("want len=%d, got %d (%v)", c.wantLen, len(got), got)
			}
		})
	}
}
