package jsonapi

import (
	"errors"
	"reflect"
	"testing"
)

const (
	atomicExtension = "https://jsonapi.org/ext/atomic"
	cursorProfile   = "http://jsonapi.org/profiles/ethanresnick/cursor-pagination/"
)

func TestCheckContentType(t *testing.T) {
	t.Parallel()

	negotiator, err := NewNegotiator(
		[]string{atomicExtension},
		[]string{cursorProfile},
	)
	if err != nil {
		t.Fatalf("create negotiator: %v", err)
	}

	tests := map[string]struct {
		header    string
		want      MediaType
		status    int
		errorCode string
	}{
		"base media type": {
			header: MediaTypeJSONAPI,
			want:   MediaType{},
		},
		"supported extension and profiles": {
			header: `application/vnd.api+json;ext="https://jsonapi.org/ext/atomic";profile="http://jsonapi.org/profiles/ethanresnick/cursor-pagination/ https://example.com/unknown"`,
			want: MediaType{
				Extensions: []string{atomicExtension},
				Profiles:   []string{cursorProfile, "https://example.com/unknown"},
			},
		},
		"missing header": {
			status:    415,
			errorCode: "unsupported-media-type",
		},
		"wrong media type": {
			header:    "application/json",
			status:    415,
			errorCode: "unsupported-media-type",
		},
		"unknown parameter": {
			header:    "application/vnd.api+json;charset=utf-8",
			status:    415,
			errorCode: "unsupported-parameter",
		},
		"unsupported extension": {
			header:    `application/vnd.api+json;ext="https://example.com/unsupported"`,
			status:    415,
			errorCode: "unsupported-extension",
		},
		"invalid extension URI": {
			header:    `application/vnd.api+json;ext="not-a-uri"`,
			status:    415,
			errorCode: "invalid-parameter",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := negotiator.CheckContentType(test.header)
			if test.status == 0 {
				if err != nil {
					t.Fatalf("check content type: %v", err)
				}
				if !reflect.DeepEqual(got, test.want) {
					t.Fatalf("unexpected media type: got %#v, want %#v", got, test.want)
				}
				return
			}

			assertNegotiationError(t, err, test.status, test.errorCode)
		})
	}
}

func TestNegotiateAccept(t *testing.T) {
	t.Parallel()

	negotiator, err := NewNegotiator(
		[]string{atomicExtension},
		[]string{cursorProfile},
	)
	if err != nil {
		t.Fatalf("create negotiator: %v", err)
	}

	tests := map[string]struct {
		header      string
		want        MediaType
		contentType string
		status      int
	}{
		"missing header accepts base": {
			contentType: MediaTypeJSONAPI,
		},
		"wildcard accepts base": {
			header:      "text/html, */*;q=0.5",
			contentType: MediaTypeJSONAPI,
		},
		"application wildcard accepts base": {
			header:      "application/*",
			contentType: MediaTypeJSONAPI,
		},
		"supported extension is selected": {
			header:      `application/vnd.api+json;ext="https://jsonapi.org/ext/atomic"`,
			want:        MediaType{Extensions: []string{atomicExtension}},
			contentType: `application/vnd.api+json; ext="https://jsonapi.org/ext/atomic"`,
		},
		"known profile is applied and unknown profile ignored": {
			header:      `application/vnd.api+json;profile="http://jsonapi.org/profiles/ethanresnick/cursor-pagination/ https://example.com/unknown"`,
			want:        MediaType{Profiles: []string{cursorProfile}},
			contentType: `application/vnd.api+json; profile="http://jsonapi.org/profiles/ethanresnick/cursor-pagination/"`,
		},
		"higher quality valid candidate wins": {
			header:      `application/vnd.api+json;profile="http://jsonapi.org/profiles/ethanresnick/cursor-pagination/";q=0.7, application/vnd.api+json;ext="https://jsonapi.org/ext/atomic";q=0.9`,
			want:        MediaType{Extensions: []string{atomicExtension}},
			contentType: `application/vnd.api+json; ext="https://jsonapi.org/ext/atomic"`,
		},
		"invalid candidate is ignored when base is available": {
			header:      "application/vnd.api+json;charset=utf-8, application/vnd.api+json",
			contentType: MediaTypeJSONAPI,
		},
		"unsupported extension candidate is ignored when base is available": {
			header:      `application/vnd.api+json;ext="https://example.com/unsupported", application/vnd.api+json;q=0.5`,
			contentType: MediaTypeJSONAPI,
		},
		"all JSON API candidates have invalid parameters": {
			header: "application/vnd.api+json;charset=utf-8",
			status: 406,
		},
		"all extension candidates are unsupported": {
			header: `application/vnd.api+json;ext="https://example.com/one", application/vnd.api+json;ext="https://example.com/two"`,
			status: 406,
		},
		"zero quality is unacceptable": {
			header: "application/vnd.api+json;q=0",
			status: 406,
		},
		"unrelated media type is unacceptable": {
			header: "application/json",
			status: 406,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := negotiator.NegotiateAccept(test.header)
			if test.status == 0 {
				if err != nil {
					t.Fatalf("negotiate accept: %v", err)
				}
				if !reflect.DeepEqual(got.MediaType, test.want) {
					t.Fatalf("unexpected media type: got %#v, want %#v", got.MediaType, test.want)
				}
				if got.ContentType != test.contentType {
					t.Fatalf("unexpected content type: got %q, want %q", got.ContentType, test.contentType)
				}
				if !got.VaryAccept {
					t.Fatal("expected negotiation to require Vary: Accept")
				}
				return
			}

			assertNegotiationError(t, err, test.status, "not-acceptable")
		})
	}
}

func TestNewNegotiatorRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	_, err := NewNegotiator([]string{"not-a-uri"}, nil)
	if err == nil {
		t.Fatal("expected configuration error")
	}
}

func assertNegotiationError(t *testing.T, err error, status int, code string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected negotiation error")
	}
	var negotiationError *NegotiationError
	if !errors.As(err, &negotiationError) {
		t.Fatalf("expected NegotiationError, got %T: %v", err, err)
	}
	if negotiationError.Status != status || negotiationError.Code != code {
		t.Fatalf(
			"unexpected error: got status %d code %q, want status %d code %q",
			negotiationError.Status,
			negotiationError.Code,
			status,
			code,
		)
	}
}
