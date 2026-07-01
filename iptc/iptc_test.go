// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package iptc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// dataset builds one IIM dataset (standard 2-byte length) for fixtures.
func dataset(record, number uint8, value string) []byte {
	b := []byte{iimTagMarker, record, number}
	b = binary.BigEndian.AppendUint16(b, uint16(len(value)))
	return append(b, value...)
}

// irb builds one Image Resource Block with an empty (even-padded) name.
func irb(resID uint16, data []byte) []byte {
	b := append([]byte(nil), irbSig...)
	b = binary.BigEndian.AppendUint16(b, resID)
	b = append(b, 0, 0) // empty Pascal name, padded to even
	b = binary.BigEndian.AppendUint32(b, uint32(len(data)))
	b = append(b, data...)
	if len(data)%2 != 0 {
		b = append(b, 0)
	}
	return b
}

// app13 assembles a full Photoshop APP13 payload from resource blocks.
func app13(blocks ...[]byte) []byte {
	out := append([]byte(nil), photoshopSig...)
	for _, b := range blocks {
		out = append(out, b...)
	}
	return out
}

// iimBlock builds a 0x0404 IPTC-IIM resource block from datasets.
func iimBlock(datasets ...[]byte) []byte {
	var iim []byte
	for _, ds := range datasets {
		iim = append(iim, ds...)
	}
	return irb(resIDIPTC, iim)
}

func TestParseRejectsNonPhotoshop(t *testing.T) {
	if _, err := Parse([]byte("not photoshop data")); err == nil {
		t.Fatal("expected error for non-Photoshop payload, got nil")
	}
}

func TestRoundTripNoChange(t *testing.T) {
	payload := app13(iimBlock(
		dataset(2, 5, "Object Name"),
		dataset(2, 80, "By-line"),
		dataset(2, 90, "Berlin"),
	))
	d, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, err := d.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !bytes.Equal(out, payload) {
		t.Fatalf("round-trip changed payload:\n in=%x\nout=%x", payload, out)
	}
}

func TestRemoveLocationDatasets(t *testing.T) {
	payload := app13(iimBlock(
		dataset(2, 5, "Object Name"),
		dataset(2, 90, "Berlin"),   // City
		dataset(2, 95, "Berlin"),   // Province/State
		dataset(2, 101, "Germany"), // Country name
		dataset(2, 80, "By-line"),
	))
	d, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for _, num := range []uint8{90, 95, 101} {
		if got := d.Remove(2, num); got != 1 {
			t.Fatalf("Remove(2,%d) = %d, want 1", num, got)
		}
	}

	out, err := d.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Kept fields survive; removed ones are gone.
	for _, keep := range []string{"Object Name", "By-line"} {
		if !bytes.Contains(out, []byte(keep)) {
			t.Errorf("expected %q to survive", keep)
		}
	}
	for _, gone := range []string{"Berlin", "Germany"} {
		if bytes.Contains(out, []byte(gone)) {
			t.Errorf("expected %q to be removed", gone)
		}
	}

	// Re-parse to confirm the rebuilt payload is well-formed and has exactly
	// the two surviving datasets.
	d2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if n := len(d2.Datasets()); n != 2 {
		t.Fatalf("re-parsed datasets = %d, want 2", n)
	}
}

func TestRemoveRepeatable(t *testing.T) {
	payload := app13(iimBlock(
		dataset(2, 26, "US-CA"), // Content Location Code (repeatable)
		dataset(2, 26, "US-NY"),
		dataset(2, 27, "California"), // Content Location Name (repeatable)
		dataset(2, 80, "By-line"),
	))
	d, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := d.Remove(2, 26); got != 2 {
		t.Fatalf("Remove(2,26) = %d, want 2", got)
	}
	if got := d.Remove(2, 27); got != 1 {
		t.Fatalf("Remove(2,27) = %d, want 1", got)
	}
	out, err := d.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	d2, _ := Parse(out)
	if n := len(d2.Datasets()); n != 1 {
		t.Fatalf("surviving datasets = %d, want 1 (By-line)", n)
	}
}

func TestBuildDropsEmptyIIMBlock(t *testing.T) {
	payload := app13(iimBlock(
		dataset(2, 90, "Berlin"),
		dataset(2, 101, "Germany"),
	))
	d, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	d.Remove(2, 90)
	d.Remove(2, 101)

	if !d.Empty() {
		t.Fatal("Empty() = false, want true after removing all datasets")
	}
	out, err := d.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Only the header should remain; no 8BIM block.
	if !bytes.Equal(out, photoshopSig) {
		t.Fatalf("expected bare header, got %x", out)
	}
}

func TestOtherBlocksPreservedVerbatim(t *testing.T) {
	thumb := irb(0x040C, []byte("THUMBNAILBYTES"))
	payload := app13(
		thumb,
		iimBlock(dataset(2, 90, "Berlin"), dataset(2, 80, "By-line")),
	)
	d, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	d.Remove(2, 90)

	if d.Empty() {
		t.Fatal("Empty() = true, want false (thumbnail block remains)")
	}
	out, err := d.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !bytes.Contains(out, thumb) {
		t.Error("thumbnail block not preserved verbatim")
	}
	if bytes.Contains(out, []byte("Berlin")) {
		t.Error("City should have been removed")
	}
}

func TestExtendedLengthDataset(t *testing.T) {
	// A value longer than the standard 2-byte length maximum forces the
	// extended-length encoding on parse and rebuild.
	big := bytes.Repeat([]byte("A"), iimStdMax+10)
	ds := []byte{iimTagMarker, 2, 80}
	ds = binary.BigEndian.AppendUint16(ds, iimExtFlag|4)
	ds = binary.BigEndian.AppendUint32(ds, uint32(len(big)))
	ds = append(ds, big...)

	payload := app13(irb(resIDIPTC, ds))
	d, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := d.Datasets()
	if len(got) != 1 || len(got[0].Value) != len(big) {
		t.Fatalf("extended dataset not parsed: %+v", got)
	}
	out, err := d.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := Parse(out); err != nil {
		t.Fatalf("rebuilt extended dataset does not re-parse: %v", err)
	}
}

func TestRemoveNoMatchReturnsZero(t *testing.T) {
	payload := app13(iimBlock(dataset(2, 80, "By-line")))
	d, _ := Parse(payload)
	if got := d.Remove(2, 90); got != 0 {
		t.Fatalf("Remove(2,90) = %d, want 0", got)
	}
}

func TestParseTruncatedDataset(t *testing.T) {
	// IIM dataset claims 100 bytes but the block is short.
	bad := []byte{iimTagMarker, 2, 90}
	bad = binary.BigEndian.AppendUint16(bad, 100)
	bad = append(bad, "short"...)
	payload := app13(irb(resIDIPTC, bad))
	if _, err := Parse(payload); err == nil {
		t.Fatal("expected error for truncated IIM dataset")
	}
}
