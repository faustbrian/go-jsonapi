module github.com/faustbrian/go-jsonapi/interoperability

go 1.27.0

require (
	github.com/DataDog/jsonapi v0.13.0
	github.com/faustbrian/go-jsonapi/v2 v2.0.0
)

require golang.org/x/text v0.41.0 // indirect

// The non-releasable harness exercises the unpublished parent source.
replace github.com/faustbrian/go-jsonapi/v2 => ..
