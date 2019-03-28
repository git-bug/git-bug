package bug

import "testing"

func TestLabelColorClassic(t *testing.T) {
	label := Label("test")
	color := label.Color()
	expected := Color{red: 244, green: 67, blue: 54}

	if color != expected {
		t.Errorf(
			"Got (R=%d, G=%d, B=%d) instead of (R=%d, G=%d, B=%d).",
			color.red, color.green, color.blue,
			expected.red, expected.green, expected.blue,
		)
	}
}

func TestLabelColorSimilar(t *testing.T) {
	label := Label("test1")
	color := label.Color()
	expected := Color{red: 121, green: 85, blue: 72}

	if color != expected {
		t.Errorf(
			"Got (R=%d, G=%d, B=%d) instead of (R=%d, G=%d, B=%d).",
			color.red, color.green, color.blue,
			expected.red, expected.green, expected.blue,
		)
	}
}

func TestLabelColorReverse(t *testing.T) {
	label := Label("tset")
	color := label.Color()
	expected := Color{red: 158, green: 158, blue: 158}

	if color != expected {
		t.Errorf(
			"Got (R=%d, G=%d, B=%d) instead of (R=%d, G=%d, B=%d).",
			color.red, color.green, color.blue,
			expected.red, expected.green, expected.blue,
		)
	}
}

func TestLabelColorEqual(t *testing.T) {
	label1 := Label("test")
	color1 := label1.Color()
	label2 := Label("test")
	color2 := label2.Color()

	if color1 != color2 {
		t.Errorf(
			"(R=%d, G=%d, B=%d) should be equal to (R=%d, G=%d, B=%d).",
			color1.red, color1.green, color1.blue,
			color2.red, color2.green, color2.blue,
		)
	}
}
