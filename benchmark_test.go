package jsonapi

import (
	"fmt"
	"testing"
)

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

func BenchmarkMarshalResourceCollection(b *testing.B) {
	document := benchmarkDocument(100, false)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Marshal(document); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalCompoundDocument(b *testing.B) {
	document := benchmarkDocument(100, true)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Marshal(document); err != nil {
			b.Fatal(err)
		}
	}
}

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

func BenchmarkMarshalAtomicOperations(b *testing.B) {
	operations := make([]AtomicOperation, 100)
	for index := range operations {
		operations[index] = AtomicOperation{
			Op:   AtomicRemove,
			Href: fmt.Sprintf("/articles/%d", index),
		}
	}
	document := AtomicDocument{Operations: operations}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := MarshalAtomic(document); err != nil {
			b.Fatal(err)
		}
	}
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
