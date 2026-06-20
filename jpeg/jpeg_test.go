// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package jpeg

import (
	"bytes"
	"testing"
)

// ---- Synthetic JPEG builders (ported from lapis strip_test.go) ----
//
// Byte-fixtures only: minimal but structurally valid segments built in code.
// No real photos live in the repo.

// appendSeg writes one marker segment (0xFF, marker, 2-byte length, payload).
func appendSeg(buf *bytes.Buffer, marker byte, data []byte) {
	buf.WriteByte(0xFF)
	buf.WriteByte(marker)
	n := uint16(len(data) + 2)
	buf.WriteByte(byte(n >> 8))
	buf.WriteByte(byte(n))
	buf.Write(data)
}

// app0 returns a minimal JFIF APP0 payload.
func app0() []byte {
	return []byte{'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 1, 0, 1, 0, 0}
}

// sof0 returns a minimal SOF0 payload (1x1 greyscale).
func sof0() []byte {
	return []byte{8, 0, 1, 0, 1, 1, 1, 0x11, 0}
}

// exifPayload returns a minimal EXIF APP1 payload: the "Exif\0\0" signature
// followed by an arbitrary (here meaningless) TIFF-ish tail. Enough to exercise
// segment identification; TIFF parsing is the exif package's concern.
func exifPayload() []byte {
	return append([]byte("Exif\x00\x00"), 'I', 'I', 42, 0, 8, 0, 0, 0)
}

// xmpPayload returns a minimal XMP APP1 payload: the Adobe xap namespace
// signature followed by a token XML body.
func xmpPayload() []byte {
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte("<x:xmpmeta/>")...)
}

// ---- Tests ----

func TestParseWriteRoundTrip(t *testing.T) {
	imageData := []byte{0x12, 0x34, 0xFF, 0x00, 0x56, 0x78} // entropy-coded tail (0xFF stuffed)
	var in bytes.Buffer
	in.Write([]byte{0xFF, 0xD8}) // SOI
	appendSeg(&in, 0xE0, app0())
	appendSeg(&in, 0xE1, exifPayload())
	appendSeg(&in, 0xE1, xmpPayload())
	appendSeg(&in, 0xC0, sof0())
	appendSeg(&in, 0xDA, []byte{0x01, 0x00}) // SOS header
	in.Write(imageData)
	in.Write([]byte{0xFF, 0xD9}) // EOI
	original := in.Bytes()

	segs, tail, err := Parse(bytes.NewReader(original))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	wantMarkers := []byte{0xE0, 0xE1, 0xE1, 0xC0, 0xDA}
	if len(segs) != len(wantMarkers) {
		t.Fatalf("got %d segments, want %d", len(segs), len(wantMarkers))
	}
	for i, m := range wantMarkers {
		if segs[i].Marker != m {
			t.Errorf("segment %d marker = 0x%02X, want 0x%02X", i, segs[i].Marker, m)
		}
	}

	// Tail is everything from the SOS payload onward: image data + EOI, verbatim.
	wantTail := append(append([]byte{}, imageData...), 0xFF, 0xD9)
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("tail = %x, want %x", tail, wantTail)
	}

	var out bytes.Buffer
	if err := Write(&out, segs, tail); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(out.Bytes(), original) {
		t.Errorf("round-trip mismatch:\n got %x\nwant %x", out.Bytes(), original)
	}
}

func TestParseStopsAtSOS(t *testing.T) {
	// Bytes after SOS that look like markers (0xFF 0xE1) must NOT be parsed as
	// segments — they are entropy-coded image data and belong in the tail.
	var in bytes.Buffer
	in.Write([]byte{0xFF, 0xD8})
	appendSeg(&in, 0xC0, sof0())
	appendSeg(&in, 0xDA, []byte{0x01, 0x00})
	trailing := []byte{0xFF, 0xE1, 0x99, 0x99, 0xFF, 0xD9} // FFE1 here is image data, not APP1
	in.Write(trailing)

	segs, tail, err := Parse(bytes.NewReader(in.Bytes()))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(segs) != 2 || segs[1].Marker != 0xDA {
		t.Fatalf("expected [SOF0, SOS], got %d segments", len(segs))
	}
	if !bytes.Equal(tail, trailing) {
		t.Errorf("tail = %x, want %x", tail, trailing)
	}
}

func TestParseSkipsFFPadding(t *testing.T) {
	// Legal 0xFF fill bytes before a marker must be skipped.
	var in bytes.Buffer
	in.Write([]byte{0xFF, 0xD8})
	in.Write([]byte{0xFF, 0xFF, 0xFF}) // padding before next marker
	appendSeg(&in, 0xC0, sof0())
	in.Write([]byte{0xFF, 0xD9})

	segs, _, err := Parse(bytes.NewReader(in.Bytes()))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(segs) != 1 || segs[0].Marker != 0xC0 {
		t.Fatalf("expected [SOF0] after padding, got %d segments", len(segs))
	}
}

func TestParseEOITailWithoutSOS(t *testing.T) {
	// A JPEG that reaches EOI before any SOS returns the standalone EOI tail.
	var in bytes.Buffer
	in.Write([]byte{0xFF, 0xD8})
	appendSeg(&in, 0xE0, app0())
	in.Write([]byte{0xFF, 0xD9})

	segs, tail, err := Parse(bytes.NewReader(in.Bytes()))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(segs) != 1 || segs[0].Marker != 0xE0 {
		t.Fatalf("expected [APP0], got %d segments", len(segs))
	}
	if !bytes.Equal(tail, []byte{0xFF, 0xD9}) {
		t.Errorf("tail = %x, want FFD9", tail)
	}
}

func TestParseRejectsNonJPEG(t *testing.T) {
	_, _, err := Parse(bytes.NewReader([]byte("this is not a jpeg")))
	if err == nil {
		t.Fatal("expected error for non-JPEG input, got nil")
	}
}

func TestParseRejectsInvalidLength(t *testing.T) {
	// Segment length of 1 is invalid (must be >= 2, covering its own field).
	in := []byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x01}
	if _, _, err := Parse(bytes.NewReader(in)); err == nil {
		t.Fatal("expected error for invalid segment length, got nil")
	}
}

func TestParseRejectsTruncatedSegment(t *testing.T) {
	// Length claims more bytes than are present.
	in := []byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0xFF}
	if _, _, err := Parse(bytes.NewReader(in)); err == nil {
		t.Fatal("expected error for truncated segment, got nil")
	}
}

func TestIsEXIF(t *testing.T) {
	cases := []struct {
		name string
		seg  Segment
		want bool
	}{
		{"exif app1", Segment{Marker: 0xE1, Data: exifPayload()}, true},
		{"xmp app1 is not exif", Segment{Marker: 0xE1, Data: xmpPayload()}, false},
		{"wrong marker", Segment{Marker: 0xE0, Data: exifPayload()}, false},
		{"short payload", Segment{Marker: 0xE1, Data: []byte("Exi")}, false},
	}
	for _, c := range cases {
		if got := IsEXIF(c.seg); got != c.want {
			t.Errorf("%s: IsEXIF = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestIsXMP(t *testing.T) {
	cases := []struct {
		name string
		seg  Segment
		want bool
	}{
		{"xmp app1", Segment{Marker: 0xE1, Data: xmpPayload()}, true},
		{"exif app1 is not xmp", Segment{Marker: 0xE1, Data: exifPayload()}, false},
		{"wrong marker", Segment{Marker: 0xE0, Data: xmpPayload()}, false},
		{"short payload", Segment{Marker: 0xE1, Data: []byte("http://")}, false},
	}
	for _, c := range cases {
		if got := IsXMP(c.seg); got != c.want {
			t.Errorf("%s: IsXMP = %v, want %v", c.name, got, c.want)
		}
	}
}
