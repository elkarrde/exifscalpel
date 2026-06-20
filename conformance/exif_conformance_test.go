// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package conformance cross-checks exifscalpel's hand-rolled EXIF engine
// against a mature, independently-maintained reference reader
// (github.com/dsoprea/go-exif/v3). It lives in its own Go module so the
// reference dependency never reaches the exifscalpel library or its
// consumers. See ../CONTRIBUTING.md "Dependency policy".
package conformance

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"codeberg.org/elkarrde/exifscalpel/exif"
	goexif "github.com/dsoprea/go-exif/v3"
)

const (
	tagMake      uint16 = 0x010F
	tagISO       uint16 = 0x8827
	tagGPSVer    uint16 = 0x0000
	tagGPSLatRef uint16 = 0x0001
)

// buildSample produces an EXIF payload via exifscalpel's own Build, carrying a
// Make, an Adobe Software signature, an ISO in the Exif sub-IFD, and a GPS
// sub-IFD — enough surface to validate parse, sub-IFD pointers, and edits.
func buildSample(t *testing.T) []byte {
	t.Helper()
	bo := binary.LittleEndian
	d := &exif.Data{
		ByteOrder: bo,
		IFD0: []exif.Entry{
			{Tag: tagMake, Type: 2, Count: 5, Value: []byte("Fuji\x00")},
			{Tag: exif.SoftwareTag, Type: 2, Count: 30, Value: append([]byte("Adobe Photoshop CS6 (Windows)"), 0)},
		},
		ExifSub: []exif.Entry{
			{Tag: tagISO, Type: 3, Count: 1, Value: []byte{0x90, 0x01}}, // ISO = 400 (LE SHORT)
		},
		GPSSub: []exif.Entry{
			{Tag: tagGPSVer, Type: 1, Count: 4, Value: []byte{2, 3, 0, 0}}, // GPSVersionID
			{Tag: tagGPSLatRef, Type: 2, Count: 2, Value: []byte{'N', 0}},  // GPSLatitudeRef
		},
	}
	payload, err := d.Build()
	if err != nil {
		t.Fatalf("exifscalpel Build: %v", err)
	}
	return payload
}

// referenceTags reads an EXIF payload with the reference library.
func referenceTags(t *testing.T, payload []byte) []goexif.ExifTag {
	t.Helper()
	raw, err := goexif.SearchAndExtractExif(payload)
	if err != nil {
		t.Fatalf("reference SearchAndExtractExif: %v", err)
	}
	tags, _, err := goexif.GetFlatExifData(raw, nil)
	if err != nil {
		t.Fatalf("reference GetFlatExifData: %v", err)
	}
	return tags
}

func findTag(tags []goexif.ExifTag, id uint16) (goexif.ExifTag, bool) {
	for _, tg := range tags {
		if tg.TagId == id {
			return tg, true
		}
	}
	return goexif.ExifTag{}, false
}

func hasGPS(tags []goexif.ExifTag) bool {
	for _, tg := range tags {
		if strings.Contains(tg.IfdPath, "GPS") {
			return true
		}
	}
	return false
}

// TestReference_ReadsOurOutput proves exifscalpel's Build emits standards-
// compliant bytes that a mature third-party reader parses correctly.
func TestReference_ReadsOurOutput(t *testing.T) {
	tags := referenceTags(t, buildSample(t))

	if mk, ok := findTag(tags, tagMake); !ok || !strings.Contains(mk.Formatted, "Fuji") {
		t.Errorf("reference Make = %q (ok=%v), want to contain Fuji", mk.Formatted, ok)
	}
	if sw, ok := findTag(tags, exif.SoftwareTag); !ok || !strings.Contains(sw.Formatted, "Adobe") {
		t.Errorf("reference Software = %q (ok=%v), want to contain Adobe", sw.Formatted, ok)
	}
	if _, ok := findTag(tags, tagISO); !ok {
		t.Error("reference did not see ISO in the Exif sub-IFD")
	}
	if !hasGPS(tags) {
		t.Error("reference did not see any GPS tags")
	}
}

// TestReference_ConfirmsScrubAndGPSRemoval proves exifscalpel's edits produce
// clean output: after RemoveIFD(GPS) + a Software scrub, the reference reader
// sees no GPS data and no Adobe signature, while Make survives.
func TestReference_ConfirmsScrubAndGPSRemoval(t *testing.T) {
	d, err := exif.Parse(buildSample(t))
	if err != nil {
		t.Fatalf("exifscalpel Parse: %v", err)
	}
	d.RemoveIFD(exif.GPSIFD)
	d.Set(exif.IFD0, exif.SoftwareTag, []byte{0}) // empty ASCII
	out, err := d.Build()
	if err != nil {
		t.Fatalf("exifscalpel Build: %v", err)
	}

	if bytes.Contains(out, []byte("Adobe")) {
		t.Error("Adobe substring survived the scrub")
	}

	tags := referenceTags(t, out)
	if hasGPS(tags) {
		t.Error("reference still sees GPS tags after RemoveIFD(GPS)")
	}
	if mk, ok := findTag(tags, tagMake); !ok || !strings.Contains(mk.Formatted, "Fuji") {
		t.Error("Make not preserved through scrub + rebuild")
	}
}
