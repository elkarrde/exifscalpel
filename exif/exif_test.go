// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package exif

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ---- Fixtures (byte-fixtures only; no real photos) ----

// buildSoftwareSeg hand-rolls an EXIF APP1 payload with a single IFD0 Software
// entry, in the given byte order. The 29-char value is stored at an offset
// (>4 bytes), exercising the external-value path. Ported from tidy-exif
// exif_test.go buildExifSeg.
func buildSoftwareSeg(order binary.ByteOrder, software string) []byte {
	val := append([]byte(software), 0) // ASCII values are NUL-terminated

	tiff := new(bytes.Buffer)
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	put16 := func(v uint16) { b := make([]byte, 2); order.PutUint16(b, v); tiff.Write(b) }
	put32 := func(v uint32) { b := make([]byte, 4); order.PutUint32(b, v); tiff.Write(b) }

	put16(42)
	put32(8) // IFD0 begins right after the 8-byte TIFF header
	put16(1) // one entry
	put16(SoftwareTag)
	put16(2)                      // type ASCII
	put32(uint32(len(val)))       // count (includes NUL)
	put32(uint32(8 + 2 + 12 + 4)) // value offset (relative to TIFF start) = 26
	put32(0)                      // no next IFD
	tiff.Write(val)

	return append(append([]byte(nil), exifSig...), tiff.Bytes()...)
}

func rational(order binary.ByteOrder, num, den uint32) []byte {
	b := make([]byte, 8)
	order.PutUint32(b[0:], num)
	order.PutUint32(b[4:], den)
	return b
}

// ---- In-place (length-preserving) Software scrub, both byte orders ----

func TestOverwriteSoftwareInPlace_BothOrders(t *testing.T) {
	const sw = "Adobe Photoshop CS6 (Windows)"
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		seg := buildSoftwareSeg(order, sw)
		orig := len(seg)

		if got := ReadValue(seg, SoftwareTag); got != sw {
			t.Fatalf("%v: ReadValue = %q, want %q", order, got, sw)
		}

		changed, err := OverwriteValueInPlace(seg, SoftwareTag, nil)
		if err != nil || !changed {
			t.Fatalf("%v: OverwriteValueInPlace: changed=%v err=%v", order, changed, err)
		}
		if len(seg) != orig {
			t.Errorf("%v: length changed %d -> %d", order, orig, len(seg))
		}
		if got := ReadValue(seg, SoftwareTag); got != "" {
			t.Errorf("%v: Software not emptied: %q", order, got)
		}
		if bytes.Contains(seg, []byte("Adobe")) {
			t.Errorf("%v: 'Adobe' substring still present", order)
		}
		// idempotent
		if changed, _ := OverwriteValueInPlace(seg, SoftwareTag, nil); changed {
			t.Errorf("%v: second overwrite reported a change", order)
		}
	}
}

func TestOverwriteSoftwareInPlace_InlineValue(t *testing.T) {
	// A short Software value (<=4 bytes) is stored inline; exercise that path.
	d := &Data{
		ByteOrder: binary.LittleEndian,
		IFD0:      []Entry{{Tag: SoftwareTag, Type: 2, Count: 3, Value: []byte("ab\x00")}},
	}
	seg, err := d.Build()
	if err != nil {
		t.Fatal(err)
	}
	if got := ReadValue(seg, SoftwareTag); got != "ab" {
		t.Fatalf("inline ReadValue = %q, want \"ab\"", got)
	}
	orig := len(seg)
	changed, err := OverwriteValueInPlace(seg, SoftwareTag, []byte("x"))
	if err != nil || !changed {
		t.Fatalf("inline overwrite: changed=%v err=%v", changed, err)
	}
	if len(seg) != orig {
		t.Errorf("inline length changed %d -> %d", orig, len(seg))
	}
	if got := ReadValue(seg, SoftwareTag); got != "x" {
		t.Errorf("inline ReadValue after = %q, want \"x\"", got)
	}
}

func TestOverwriteValueInPlace_AbsentAndInvalid(t *testing.T) {
	seg := buildSoftwareSeg(binary.LittleEndian, "VueScan x64 9.5.60")

	// Absent tag: no change, no error.
	if changed, err := OverwriteValueInPlace(seg, 0x9999, nil); changed || err != nil {
		t.Errorf("absent tag: changed=%v err=%v, want false/nil", changed, err)
	}
	// Not EXIF: error.
	if _, err := OverwriteValueInPlace([]byte("not exif at all"), SoftwareTag, nil); err == nil {
		t.Error("expected error for non-EXIF payload")
	}
}

// ---- Rebuild-mode Software scrub ----

func TestSetSoftware_Rebuild(t *testing.T) {
	seg := buildSoftwareSeg(binary.LittleEndian, "Adobe Photoshop CS6 (Windows)")
	d, err := Parse(seg)
	if err != nil {
		t.Fatal(err)
	}
	d.Set(IFD0, SoftwareTag, []byte{0}) // empty ASCII (single NUL)
	out, err := d.Build()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("Adobe")) {
		t.Error("rebuild: 'Adobe' substring still present")
	}
	d2, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := d2.Find(IFD0, SoftwareTag); !ok {
		t.Error("rebuild: Software entry unexpectedly gone")
	} else if v := string(bytes.TrimRight(e.Value, "\x00")); v != "" {
		t.Errorf("rebuild: Software not emptied: %q", v)
	}
}

// ---- lapis behavior: GPS IFD removal (ported) ----

func TestRemoveGPSIFD(t *testing.T) {
	bo := binary.LittleEndian
	d := &Data{
		ByteOrder: bo,
		IFD0:      []Entry{{Tag: 0x010F, Type: 2, Count: 5, Value: []byte("Test\x00")}}, // Make
		GPSSub: []Entry{
			{Tag: 0x0001, Type: 2, Count: 2, Value: []byte{'N', 0}},                                                                     // GPSLatitudeRef
			{Tag: 0x0002, Type: 5, Count: 3, Value: append(append(rational(bo, 51, 1), rational(bo, 30, 1)...), rational(bo, 0, 1)...)}, // GPSLatitude
		},
	}
	seg, err := d.Build()
	if err != nil {
		t.Fatal(err)
	}

	// Sanity: parsed back, GPS is present and the pointer was synthesized.
	p, err := Parse(seg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Find(IFD0, GPSIFDPointer); !ok {
		t.Fatal("setup: GPS pointer missing after Build")
	}
	if len(p.GPSSub) == 0 {
		t.Fatal("setup: GPS sub-IFD missing after parse")
	}

	// Remove GPS wholesale, rebuild, re-parse.
	p.RemoveIFD(GPSIFD)
	out, err := p.Build()
	if err != nil {
		t.Fatal(err)
	}
	q, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := q.Find(IFD0, GPSIFDPointer); ok {
		t.Error("GPS IFD pointer still present in IFD0")
	}
	if len(q.GPSSub) != 0 {
		t.Error("GPS sub-IFD still present")
	}
	// Make must be preserved.
	if _, ok := q.Find(IFD0, 0x010F); !ok {
		t.Error("Make tag unexpectedly removed")
	}
}

// ---- Parse/edit basics + round-trip stability ----

func TestParseBuildRoundTrip(t *testing.T) {
	bo := binary.BigEndian
	d := &Data{
		ByteOrder: bo,
		IFD0: []Entry{
			{Tag: 0x0100, Type: 4, Count: 1, Value: []byte{0, 0, 0, 1}}, // ImageWidth (inline)
			{Tag: 0x010F, Type: 2, Count: 5, Value: []byte("Fuji\x00")}, // Make (offset)
		},
		ExifSub: []Entry{
			{Tag: 0x8827, Type: 3, Count: 1, Value: []byte{0x01, 0x90}}, // ISO=400 (inline SHORT)
		},
	}
	seg, err := d.Build()
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse(seg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Find(IFD0, ExifIFDPointer); !ok {
		t.Error("Exif sub-IFD pointer not synthesized")
	}
	if e, ok := p.Find(ExifIFD, 0x8827); !ok {
		t.Error("ISO missing from Exif sub-IFD")
	} else if bo.Uint16(e.Value) != 400 {
		t.Errorf("ISO = %d, want 400", bo.Uint16(e.Value))
	}
	if e, ok := p.Find(IFD0, 0x010F); !ok || string(bytes.TrimRight(e.Value, "\x00")) != "Fuji" {
		t.Errorf("Make round-trip failed: %v / %q", ok, e)
	}
}

func TestFindSetRemove(t *testing.T) {
	d := &Data{
		ByteOrder: binary.LittleEndian,
		IFD0: []Entry{
			{Tag: 0x010F, Type: 2, Count: 5, Value: []byte("Test\x00")},
		},
	}
	if _, ok := d.Find(IFD0, 0x010F); !ok {
		t.Fatal("Find: Make not found")
	}
	// Set existing updates value + count.
	d.Set(IFD0, 0x010F, []byte("Hi\x00"))
	if e, _ := d.Find(IFD0, 0x010F); e.Count != 3 || string(e.Value) != "Hi\x00" {
		t.Errorf("Set existing: count=%d value=%q", e.Count, e.Value)
	}
	// Set absent appends ASCII entry.
	d.Set(IFD0, 0x0131, []byte("x\x00"))
	if e, ok := d.Find(IFD0, 0x0131); !ok || e.Type != 2 {
		t.Errorf("Set absent: ok=%v type=%d", ok, e.Type)
	}
	// Remove.
	d.Remove(IFD0, 0x010F)
	if _, ok := d.Find(IFD0, 0x010F); ok {
		t.Error("Remove: Make still present")
	}
}

func TestParseRejectsBadPayload(t *testing.T) {
	cases := map[string][]byte{
		"no exif sig": []byte("not exif"),
		"bad order":   append([]byte("Exif\x00\x00"), 'X', 'X', 0, 42, 0, 0, 0, 8),
		"short":       []byte("Exif\x00\x00II"),
	}
	for name, in := range cases {
		if _, err := Parse(in); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}
