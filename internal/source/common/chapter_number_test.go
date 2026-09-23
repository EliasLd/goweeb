package common

import "testing"

func TestParseChapterNumber(t *testing.T) {
	valid := map[string]string{"0": "0", "00.000": "0", "06.500": "6.5", "14.2500": "14.25", "0010": "10", "1.01": "1.01"}
	for input, want := range valid {
		number, err := ParseChapterNumber(input)
		if err != nil || number.String() != want {
			t.Errorf("%q: got %q err %v want %q", input, number, err, want)
		}
	}
	for _, input := range []string{"", "-1", ".5", "1.", "1..2", "1.2a", "+1"} {
		if _, err := ParseChapterNumber(input); err == nil {
			t.Errorf("expected %q to be invalid", input)
		}
	}
}
func TestNumberCompareAndPadding(t *testing.T) {
	ordered := []string{"0", "0.5", "1", "1.01", "1.1", "6.5", "7", "9.9", "10", "14.25"}
	for i := 0; i < len(ordered)-1; i++ {
		a, _ := ParseChapterNumber(ordered[i])
		b, _ := ParseChapterNumber(ordered[i+1])
		if a.Compare(b) >= 0 {
			t.Errorf("expected %s < %s", a, b)
		}
	}
	a, _ := ParseChapterNumber("6.5")
	if a.Padded(3) != "006.5" {
		t.Fatalf("got %q", a.Padded(3))
	}
}
