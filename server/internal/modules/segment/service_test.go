package segment

import "testing"

func TestNormalizeListRequestCapsLimitAtMaximum(t *testing.T) {
	req := ListRequest{Limit: 150, Offset: -1}

	normalizeListRequest(&req)

	if req.Limit != 100 {
		t.Fatalf("limit = %d, want 100", req.Limit)
	}
	if req.Offset != 0 {
		t.Fatalf("offset = %d, want 0", req.Offset)
	}
}

func TestNormalizeListRequestDefaultsNonPositiveLimit(t *testing.T) {
	req := ListRequest{Limit: 0, Offset: 10}

	normalizeListRequest(&req)

	if req.Limit != 50 {
		t.Fatalf("limit = %d, want 50", req.Limit)
	}
	if req.Offset != 10 {
		t.Fatalf("offset = %d, want 10", req.Offset)
	}
}
