package bug

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelRGBA(t *testing.T) {
	rgba := Label("test").Color()
	expected := LabelColor{R: 255, G: 87, B: 34, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBASimilar(t *testing.T) {
	rgba := Label("test1").Color()
	expected := LabelColor{R: 0, G: 188, B: 212, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAReverse(t *testing.T) {
	rgba := Label("tset").Color()
	expected := LabelColor{R: 233, G: 30, B: 99, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAEqual(t *testing.T) {
	color1 := Label("test").Color()
	color2 := Label("test").Color()

	require.Equal(t, color1, color2)
}
