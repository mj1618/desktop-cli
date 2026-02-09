package platform

import "testing"

func TestParseBBox_Valid(t *testing.T) {
	b, err := ParseBBox("10,20,300,400")
	if err != nil {
		t.Fatal(err)
	}
	if b.X != 10 || b.Y != 20 || b.Width != 300 || b.Height != 400 {
		t.Errorf("got %+v, want {10 20 300 400}", b)
	}
}

func TestParseBBox_WithSpaces(t *testing.T) {
	b, err := ParseBBox("10, 20, 300, 400")
	if err != nil {
		t.Fatal(err)
	}
	if b.X != 10 || b.Y != 20 || b.Width != 300 || b.Height != 400 {
		t.Errorf("got %+v, want {10 20 300 400}", b)
	}
}

func TestParseBBox_Invalid(t *testing.T) {
	tests := []string{
		"",
		"10,20,300",
		"10,20,300,400,500",
		"a,b,c,d",
		"10,20,abc,400",
	}
	for _, s := range tests {
		_, err := ParseBBox(s)
		if err == nil {
			t.Errorf("ParseBBox(%q) should fail", s)
		}
	}
}

func TestParseMouseButton_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  MouseButton
	}{
		{"left", MouseLeft},
		{"Left", MouseLeft},
		{"LEFT", MouseLeft},
		{"right", MouseRight},
		{"Right", MouseRight},
		{"middle", MouseMiddle},
		{"Middle", MouseMiddle},
	}
	for _, tt := range tests {
		got, err := ParseMouseButton(tt.input)
		if err != nil {
			t.Errorf("ParseMouseButton(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ParseMouseButton(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseMouseButton_Invalid(t *testing.T) {
	_, err := ParseMouseButton("invalid")
	if err == nil {
		t.Error("ParseMouseButton(\"invalid\") should fail")
	}
}
