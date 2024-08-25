package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelRGBA(t *testing.T) {
	rgba := Label("test1").Color()
	expected := LabelColor{R: 0, G: 150, B: 136, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBASimilar(t *testing.T) {
	rgba := Label("test2").Color()
	expected := LabelColor{R: 3, G: 169, B: 244, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAReverse(t *testing.T) {
	rgba := Label("tset").Color()
	expected := LabelColor{R: 63, G: 81, B: 181, A: 255}

	require.Equal(t, expected, rgba)
}

func TestLabelRGBAEqual(t *testing.T) {
	color1 := Label("test").Color()
	color2 := Label("test").Color()

	require.Equal(t, color1, color2)
}
