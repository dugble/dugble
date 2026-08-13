package sms

import (
	"strings"
	"testing"
)

func TestAnalyzeBodyReportsEncodingAndSegments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, body, encoding string
		segments             int32
	}{
		{name: "gsm", body: "hello", encoding: "GSM-7", segments: 1},
		{name: "unicode", body: "hello 👋", encoding: "UCS-2", segments: 1},
		{name: "multipart", body: strings.Repeat("a", 161), encoding: "GSM-7", segments: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoding, _, segments := AnalyzeBody(test.body)
			if encoding != test.encoding || segments != test.segments {
				t.Fatalf("AnalyzeBody() = (%q, %d), want (%q, %d)", encoding, segments, test.encoding, test.segments)
			}
		})
	}
}
