package jsonapi

import (
	"fmt"
	"net/url"
	"testing"
)

// BenchmarkMarshalSingleResource includes checked encoding for one resource;
// fixture construction is outside b.Loop, not part of the measured operation.
func BenchmarkMarshalSingleResource(b *testing.B) {
	collection := benchmarkDocument(1, false)
	document := Document{Data: ResourceData(collection.Data.many[0])}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Marshal(document); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalResourceCollection measures checked encoding of a prepared
// 100-resource collection, without document construction or application I/O.
func BenchmarkMarshalResourceCollection(b *testing.B) {
	document := benchmarkDocument(100, false)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Marshal(document); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalSingleResource measures full checked decoding from bytes;
// preparing its one-resource payload is outside the measured loop.
func BenchmarkUnmarshalSingleResource(b *testing.B) {
	document := Document{Data: ResourceData(benchmarkDocument(1, false).Data.many[0])}
	payload, err := Marshal(document)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		if _, err := Unmarshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalResourceCollection measures admission and decoding of a
// prepared 100-resource payload; bytes/allocation figures exclude preparation.
func BenchmarkUnmarshalResourceCollection(b *testing.B) {
	payload, err := Marshal(benchmarkDocument(100, false))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		if _, err := Unmarshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalCompoundDocument measures encoding and linkage validation
// for 100 primary resources and their included authors, with prepared input.
func BenchmarkMarshalCompoundDocument(b *testing.B) {
	document := benchmarkDocument(100, true)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Marshal(document); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalCompoundDocument measures full decoding and linkage checks
// for the prepared compound fixture, excluding its initial encoding.
func BenchmarkUnmarshalCompoundDocument(b *testing.B) {
	payload, err := Marshal(benchmarkDocument(100, true))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		if _, err := Unmarshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalAdversarialCompoundDocument stresses cyclic included
// linkage and ID/LID aliases across 1,000 authors; fixture setup is excluded.
func BenchmarkUnmarshalAdversarialCompoundDocument(b *testing.B) {
	payload, err := Marshal(benchmarkAdversarialCompoundDocument(1_000))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		if _, err := Unmarshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalAtomicOperations measures checked encoding of 100 prepared
// remove operations; application transaction execution is excluded.
func BenchmarkMarshalAtomicOperations(b *testing.B) {
	document := benchmarkAtomicDocument(100)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := MarshalAtomic(document); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalAtomicOperations measures full checked decoding of 100
// remove operations from prepared bytes, not executing them.
func BenchmarkUnmarshalAtomicOperations(b *testing.B) {
	payload, err := MarshalAtomic(benchmarkAtomicDocument(100))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		if _, err := UnmarshalAtomic(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseQuery measures family parsing of a fixed decoded query; URL
// decoding, HTTP admission and applying filters/ordering are excluded.
func BenchmarkParseQuery(b *testing.B) {
	values := url.Values{
		"include":          {"author.comments,tags"},
		"fields[articles]": {"title,body,createdAt"},
		"filter[status]":   {"published"},
		"page[after]":      {"opaque-cursor"},
		"page[size]":       {"50"},
		"sort":             {"-createdAt,title"},
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseQuery(values); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNegotiateAccept measures selection for one fixed header/capability
// set; registry construction and actually applying profiles are excluded.
func BenchmarkNegotiateAccept(b *testing.B) {
	const extension = "https://example.com/extensions/version"
	const profile = "https://example.com/profiles/timestamps"
	negotiator, err := NewNegotiator([]string{extension}, []string{profile})
	if err != nil {
		b.Fatal(err)
	}
	header := "application/json;q=0.2, " + MediaTypeJSONAPI +
		`;ext="` + extension + `";profile="` + profile + `";q=0.9`
	b.ReportAllocs()
	for b.Loop() {
		if _, err := negotiator.NegotiateAccept(header); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCursorPaginationMetadata measures wrapping and parsing fixed count
// metadata together; it does not query or estimate a real collection.
func BenchmarkCursorPaginationMetadata(b *testing.B) {
	total := int64(100_000)
	estimate := int64(100_500)
	truncated := true
	metadata := CursorPageMeta{
		RangeTruncated: &truncated,
		Total:          &total,
		EstimatedTotal: &CursorEstimatedTotal{BestGuess: &estimate},
	}
	b.ReportAllocs()
	for b.Loop() {
		meta, err := metadata.Meta()
		if err != nil {
			b.Fatal(err)
		}
		if _, _, err := ParseCursorPageMeta(meta); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkAtomicDocument(size int) AtomicDocument {
	operations := make([]AtomicOperation, size)
	for index := range operations {
		operations[index] = AtomicOperation{
			Op:   AtomicRemove,
			Href: fmt.Sprintf("/articles/%d", index),
		}
	}
	return AtomicDocument{Operations: operations}
}

func benchmarkDocument(size int, compound bool) Document {
	resources := make([]ResourceObject, size)
	included := make([]ResourceObject, 0, size)
	for index := range resources {
		id := fmt.Sprintf("%d", index)
		resource := ResourceObject{
			Type: "articles",
			ID:   id,
			Attributes: Attributes{
				"title": fmt.Sprintf("Article %d", index),
				"body":  "Representative benchmark content",
			},
		}
		if compound {
			authorID := "author-" + id
			resource.Relationships = Relationships{
				"author": {Data: ToOne(Identifier{Type: "people", ID: authorID})},
			}
			included = append(included, ResourceObject{
				Type:       "people",
				ID:         authorID,
				Attributes: Attributes{"name": "Benchmark Author"},
			})
		}
		resources[index] = resource
	}
	document := Document{Data: ResourceCollection(resources...)}
	if compound {
		document.Included = included
	}
	return document
}

func benchmarkAdversarialCompoundDocument(size int) Document {
	document := benchmarkDocument(size, true)
	for index := range document.Included {
		id := document.Included[index].ID
		next := document.Included[(index+1)%len(document.Included)].ID
		document.Included[index].LID = "local-" + id
		document.Included[index].Relationships = Relationships{
			"next": {Data: ToOne(Identifier{Type: "people", ID: next})},
		}
	}
	return document
}
