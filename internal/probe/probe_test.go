package probe

import "testing"

func TestParseColo(t *testing.T) {
	if got := parseColo("fl=1\ncolo=HKG\n", "abc-NRT"); got != "HKG" {
		t.Fatalf("got %q", got)
	}
	if got := parseColo("", "abc-AMS"); got != "AMS" {
		t.Fatalf("got %q", got)
	}
}
