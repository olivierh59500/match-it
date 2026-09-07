package resources

import "testing"

func TestRuntimeAssetsAreEmbedded(t *testing.T) {
	for _, name := range []string{
		"png/gamearea.img.png",
		"png/tile_00.png",
		"png/font2/glyph_00.png",
		"png/remove/removean.img.png",
		"music/Chambers of Shaolin - Trapped in China.ym",
	} {
		data, err := Files.ReadFile(name)
		if err != nil {
			t.Errorf("read embedded %q: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded %q is empty", name)
		}
	}
}
