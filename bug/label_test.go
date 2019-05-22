package bug

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelRGBA(t *testing.T) {
	rgba := Label("test").RGBA()
	expected := color.RGBA{R: 255, G: 87, B: 34, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBASimilar(t *testing.T) {
	rgba := Label("test1").RGBA()
	expected := color.RGBA{R: 0, G: 188, B: 212, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAReverse(t *testing.T) {
	rgba := Label("tset").RGBA()
	expected := color.RGBA{R: 233, G: 30, B: 99, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAEqual(t *testing.T) {
	color1 := Label("test").RGBA()
	color2 := Label("test").RGBA()

	require.Equal(t, color1, color2)
}
