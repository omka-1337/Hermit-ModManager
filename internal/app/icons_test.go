package app

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// makeICO builds an .ico with the given images, each described by width and data.
func makeICO(images []struct {
	width int
	data  []byte
}) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, []uint16{0, 1, uint16(len(images))})
	offset := 6 + 16*len(images)
	for _, img := range images {
		w := byte(img.width)
		if img.width == 256 {
			w = 0
		}
		buf.Write([]byte{w, w, 0, 0, 1, 0, 32, 0})
		binary.Write(&buf, binary.LittleEndian, []uint32{uint32(len(img.data)), uint32(offset)})
		offset += len(img.data)
	}
	for _, img := range images {
		buf.Write(img.data)
	}
	return buf.Bytes()
}

func TestLargestPNGInICO(t *testing.T) {
	png := func(tag string) []byte { return append([]byte("\x89PNG\r\n\x1a\n"), tag...) }
	ico := makeICO([]struct {
		width int
		data  []byte
	}{
		{32, png("small")},
		{256, png("large")},
		{48, []byte("BMP bitmap")},
	})
	if got := largestPNGInICO(ico); !bytes.HasSuffix(got, []byte("large")) {
		t.Errorf("got %q", got)
	}
	bitmaps := makeICO([]struct {
		width int
		data  []byte
	}{{32, []byte("BMP")}})
	if largestPNGInICO(bitmaps) != nil || largestPNGInICO([]byte("junk")) != nil {
		t.Error("expected nil without PNG images")
	}
}
