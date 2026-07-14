package jsonapi

import (
	"net/url"
	"reflect"
	"testing"
)

func FuzzUnmarshal(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"data":null}`),
		[]byte(`{"data":[]}`),
		[]byte(`{"data":{"type":"articles","id":"1"}}`),
		[]byte(`{"errors":[{"status":"400","title":"Bad request"}]}`),
		[]byte(`{"data":null,"data":[]}`),
		[]byte(`{"data":`),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		document, err := Unmarshal(payload)
		if err != nil {
			return
		}
		canonical, err := Marshal(document)
		if err != nil {
			t.Fatalf("accepted document cannot be marshaled: %v", err)
		}
		if _, err := Unmarshal(canonical); err != nil {
			t.Fatalf("canonical document cannot be decoded: %v", err)
		}
	})
}

func FuzzUnmarshalAtomic(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"atomic:operations":[{"op":"remove","href":"/articles/1"}]}`),
		[]byte(`{"atomic:results":[{}]}`),
		[]byte(`{"errors":[{"status":"409"}]}`),
		[]byte(`{"data":null}`),
		[]byte(`{"atomic:operations":[{"op":"add"}]}`),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		document, err := UnmarshalAtomic(payload)
		if err != nil {
			return
		}
		canonical, err := MarshalAtomic(document)
		if err != nil {
			t.Fatalf("accepted Atomic document cannot be marshaled: %v", err)
		}
		if _, err := UnmarshalAtomic(canonical); err != nil {
			t.Fatalf("canonical Atomic document cannot be decoded: %v", err)
		}
	})
}

func FuzzParseQuery(f *testing.F) {
	seeds := []string{
		"include=author.comments&fields%5Barticles%5D=title,body&sort=-createdAt",
		"page%5Bsize%5D=25&page%5Bafter%5D=opaque&filter%5Bstatus%5D=published",
		"include=%",
		"fields%5B%5D=title",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, rawQuery string) {
		values, err := url.ParseQuery(rawQuery)
		if err != nil {
			return
		}
		first, err := ParseQuery(values)
		if err != nil {
			return
		}
		second, err := ParseQuery(values)
		if err != nil {
			t.Fatalf("accepted query cannot be parsed again: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("query parsing is not deterministic: %#v != %#v", first, second)
		}
	})
}

func FuzzCursorPaginationQuery(f *testing.F) {
	pagination, err := NewCursorPagination(CursorPaginationConfig{
		DefaultSize: 20,
		MaxSize:     100,
		AllowRange:  true,
	})
	if err != nil {
		f.Fatal(err)
	}
	seeds := []string{
		"page%5Bsize%5D=25&page%5Bafter%5D=opaque",
		"page%5Bbefore%5D=older",
		"page%5Bafter%5D=a&page%5Bbefore%5D=b",
		"page%5Bsize%5D=-1",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, rawQuery string) {
		values, err := url.ParseQuery(rawQuery)
		if err != nil {
			return
		}
		query, err := ParseQuery(values)
		if err != nil {
			return
		}
		request, err := pagination.ParseQuery(query)
		if err != nil {
			return
		}
		if request.Size < 1 || request.Size > 100 {
			t.Fatalf("accepted page size is outside configured bounds: %d", request.Size)
		}
	})
}

func FuzzNegotiation(f *testing.F) {
	const extension = "https://example.com/extensions/version"
	const profile = "https://example.com/profiles/timestamps"
	negotiator, err := NewNegotiator([]string{extension}, []string{profile})
	if err != nil {
		f.Fatal(err)
	}
	seeds := []string{
		MediaTypeJSONAPI,
		MediaTypeJSONAPI + `;ext="` + extension + `"`,
		MediaTypeJSONAPI + `;profile="` + profile + `"`,
		"application/json, */*;q=0.5",
		"not a media type",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, header string) {
		if mediaType, err := negotiator.CheckContentType(header); err == nil {
			if _, err := negotiator.CheckContentType(mediaType.String()); err != nil {
				t.Fatalf("canonical accepted content type was rejected: %v", err)
			}
		}
		if selected, err := negotiator.NegotiateAccept(header); err == nil {
			if _, err := negotiator.CheckContentType(selected.ContentType); err != nil {
				t.Fatalf("negotiated content type was rejected: %v", err)
			}
		}
	})
}
