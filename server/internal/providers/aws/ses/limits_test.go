package ses

import "testing"

func TestPayloadLimits(t *testing.T) {
	tests := map[string]struct {
		got  int
		want int
	}{
		"raw message":         {got: MaxRawMessageBytes, want: 10 << 20},
		"body":                {got: MaxBodyBytes, want: 1 << 20},
		"attachments decoded": {got: MaxAttachmentsDecodedBytes, want: 7 << 20},
		"batch payload":       {got: MaxBatchPayloadBytes, want: 10 << 20},
		"http request":        {got: MaxHTTPRequestBytes, want: 12 << 20},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got %d bytes, want %d", test.got, test.want)
			}
		})
	}
}
