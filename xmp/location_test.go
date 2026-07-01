// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmp

import (
	"bytes"
	"testing"
)

// flatLocationXMP carries GPS coordinates and place-names as attributes on the
// rdf:Description, plus a couple in element form.
const flatLocationXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:exif="http://ns.adobe.com/exif/1.0/"
        xmlns:photoshop="http://ns.adobe.com/photoshop/1.0/"
        xmlns:Iptc4xmpCore="http://iptc.org/std/Iptc4xmpCore/1.0/xmlns/"
        exif:GPSLatitude="52,31.5N"
        exif:GPSLongitude="13,24.3E"
        exif:GPSAltitude="34/1"
        photoshop:City="Berlin"
        photoshop:Country="Germany"
        Iptc4xmpCore:CountryCode="DE">
      <photoshop:State>Berlin</photoshop:State>
      <Iptc4xmpCore:Location>Alexanderplatz</Iptc4xmpCore:Location>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

// structuredLocationXMP nests location leaves inside Iptc4xmpExt:LocationCreated
// and LocationShown (rdf:Bag of location structures).
const structuredLocationXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:Iptc4xmpExt="http://iptc.org/std/Iptc4xmpExt/2008-02-29/"
        xmlns:exif="http://ns.adobe.com/exif/1.0/">
      <Iptc4xmpExt:LocationCreated>
        <rdf:Bag>
          <rdf:li rdf:parseType="Resource">
            <Iptc4xmpExt:City>Reykjavik</Iptc4xmpExt:City>
            <Iptc4xmpExt:CountryName>Iceland</Iptc4xmpExt:CountryName>
            <Iptc4xmpExt:CountryCode>IS</Iptc4xmpExt:CountryCode>
            <Iptc4xmpExt:Sublocation>Downtown</Iptc4xmpExt:Sublocation>
            <exif:GPSLatitude>64,08.5N</exif:GPSLatitude>
          </rdf:li>
        </rdf:Bag>
      </Iptc4xmpExt:LocationCreated>
      <Iptc4xmpExt:LocationShown>
        <rdf:Bag>
          <rdf:li rdf:parseType="Resource">
            <Iptc4xmpExt:City>Vik</Iptc4xmpExt:City>
            <Iptc4xmpExt:WorldRegion>Europe</Iptc4xmpExt:WorldRegion>
          </rdf:li>
        </rdf:Bag>
      </Iptc4xmpExt:LocationShown>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

// noLocationXMP has only Adobe-signature fields (Clean's territory), no location.
const noLocationXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:xmp="http://ns.adobe.com/xap/1.0/"
        xmp:CreatorTool="Adobe Lightroom Classic 13.0"/>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

func TestCleanLocationFlat(t *testing.T) {
	payload := makeSeg(flatLocationXMP)
	out, changed, err := CleanLocation(payload)
	if err != nil {
		t.Fatalf("CleanLocation: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if len(out) != len(payload) {
		t.Fatalf("length not preserved: got %d, want %d", len(out), len(payload))
	}
	for _, leak := range []string{"52,31.5N", "13,24.3E", "34/1", "Berlin", "Germany", "DE", "Alexanderplatz"} {
		if bytes.Contains(out, []byte(leak)) {
			t.Errorf("location value %q survived", leak)
		}
	}
}

func TestCleanLocationStructured(t *testing.T) {
	payload := makeSeg(structuredLocationXMP)
	out, changed, err := CleanLocation(payload)
	if err != nil {
		t.Fatalf("CleanLocation: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if len(out) != len(payload) {
		t.Fatalf("length not preserved: got %d, want %d", len(out), len(payload))
	}
	for _, leak := range []string{"Reykjavik", "Iceland", "IS", "Downtown", "64,08.5N", "Vik", "Europe"} {
		if bytes.Contains(out, []byte(leak)) {
			t.Errorf("structured location value %q survived", leak)
		}
	}
	// The containers themselves remain, keeping the XML well-formed.
	for _, keep := range []string{"Iptc4xmpExt:LocationCreated", "Iptc4xmpExt:LocationShown", "rdf:Bag"} {
		if !bytes.Contains(out, []byte(keep)) {
			t.Errorf("expected container %q to remain", keep)
		}
	}
}

func TestCleanLocationNoLocationUnchanged(t *testing.T) {
	payload := makeSeg(noLocationXMP)
	out, changed, err := CleanLocation(payload)
	if err != nil {
		t.Fatalf("CleanLocation: %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false for payload with no location data")
	}
	if !bytes.Equal(out, payload) {
		t.Fatal("payload with no location data should be returned unchanged")
	}
}

func TestCleanLocationRejectsNonXMP(t *testing.T) {
	if _, _, err := CleanLocation([]byte("not xmp")); err == nil {
		t.Fatal("expected error for non-XMP payload")
	}
}

// TestCleanLocationLeavesAdobeFields confirms CleanLocation does not touch the
// Adobe-signature fields that the default Clean targets — the two entry points
// stay independent (tidy-exif's Clean behavior is unaffected).
func TestCleanLocationLeavesAdobeFields(t *testing.T) {
	payload := makeSeg(noLocationXMP)
	out, _, err := CleanLocation(payload)
	if err != nil {
		t.Fatalf("CleanLocation: %v", err)
	}
	if !bytes.Contains(out, []byte("Adobe Lightroom Classic 13.0")) {
		t.Error("CleanLocation should not remove Adobe CreatorTool")
	}
}
