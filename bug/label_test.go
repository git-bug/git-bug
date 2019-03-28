package bug

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelRGBColor(t *testing.T) {
	color := Label("test").RGBColor()
	expected := RGBColor{red: 255, green: 87, blue: 34}

	require.Equal(t, expected, color)
}

func TestLabelRGBColorSimilar(t *testing.T) {
	color := Label("test1").RGBColor()
	expected := RGBColor{red: 0, green: 188, blue: 212}

	require.Equal(t, expected, color)
}

func TestLabelRGBColorReverse(t *testing.T) {
	color := Label("tset").RGBColor()
	expected := RGBColor{red: 233, green: 30, blue: 99}

	require.Equal(t, expected, color)
}

func TestLabelRGBColorEqual(t *testing.T) {
	color1 := Label("test").RGBColor()
	color2 := Label("test").RGBColor()

	require.Equal(t, color1, color2)
}
