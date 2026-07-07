// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmp

import "testing"

const (
	nsAnalogExif = "http://analogexif.sourceforge.net/ns/"
	nsAux        = "http://ns.adobe.com/exif/1.0/aux/"
)

// analogExifPayload mirrors a real ExifNotes/AnalogExif packet: scalars in
// attribute form on rdf:Description (Film, FilmMaker, aux:Lens) plus one field
// in element form (Developer), with an XML entity to decode (B&amp;W).
func analogExifPayload() []byte {
	xml := `<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>` +
		`<x:xmpmeta xmlns:x='adobe:ns:meta/'>` +
		`<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>` +
		`<rdf:Description rdf:about=''` +
		` xmlns:AnalogExif='http://analogexif.sourceforge.net/ns/'` +
		` xmlns:aux='http://ns.adobe.com/exif/1.0/aux/'` +
		` AnalogExif:Film='Mono' AnalogExif:FilmMaker='KosmoFoto'` +
		` AnalogExif:DevelopProcess='B&amp;W'` +
		` aux:Lens='Tokina SZ-X 80-200mm F4.5-5.6'>` +
		`<AnalogExif:Developer>Adonal</AnalogExif:Developer>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta><?xpacket end='w'?>`
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(xml)...)
}

func TestReadProperties(t *testing.T) {
	want := []Property{
		{nsAnalogExif, "Film"},           // attribute form
		{nsAnalogExif, "FilmMaker"},      // attribute form
		{nsAnalogExif, "DevelopProcess"}, // attribute form with entity
		{nsAnalogExif, "Developer"},      // element form
		{nsAux, "Lens"},                  // attribute form, different namespace
		{nsAnalogExif, "Missing"},        // absent -> omitted
	}
	got, err := ReadProperties(analogExifPayload(), want)
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}

	expect := map[Property]string{
		{nsAnalogExif, "Film"}:           "Mono",
		{nsAnalogExif, "FilmMaker"}:      "KosmoFoto",
		{nsAnalogExif, "DevelopProcess"}: "B&W",
		{nsAnalogExif, "Developer"}:      "Adonal",
		{nsAux, "Lens"}:                  "Tokina SZ-X 80-200mm F4.5-5.6",
	}
	for p, v := range expect {
		if got[p] != v {
			t.Errorf("%s:%s = %q, want %q", p.Namespace, p.Name, got[p], v)
		}
	}
	if _, ok := got[Property{nsAnalogExif, "Missing"}]; ok {
		t.Error("absent property should be omitted from the result")
	}
	if len(got) != len(expect) {
		t.Errorf("got %d properties, want %d: %v", len(got), len(expect), got)
	}
}

// TestReadPropertiesNamespaceIsolation confirms matching is by namespace URI,
// not just local name: a different namespace's same-named field must not match.
func TestReadPropertiesNamespaceIsolation(t *testing.T) {
	got, err := ReadProperties(analogExifPayload(), []Property{{nsAux, "Film"}})
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}
	if v, ok := got[Property{nsAux, "Film"}]; ok {
		t.Errorf("aux:Film should not match AnalogExif:Film, got %q", v)
	}
}

// TestReadPropertiesNotXMP guards the signature check.
func TestReadPropertiesNotXMP(t *testing.T) {
	if _, err := ReadProperties([]byte("not xmp"), []Property{{nsAux, "Lens"}}); err == nil {
		t.Error("expected an error for a non-XMP payload")
	}
}
