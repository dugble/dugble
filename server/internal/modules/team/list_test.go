package team

import "testing"

func TestParseListOptionsDefaults(t *testing.T) {
	options, err := parseListOptions("", "", "", "")
	if err != nil {
		t.Fatalf("parseListOptions returned error: %v", err)
	}
	if options.Page != 1 || options.Limit != 20 || options.Search != "" || options.Status != TeamStatusActive {
		t.Fatalf("unexpected defaults: %#v", options)
	}
}

func TestParseListOptionsNormalizesValues(t *testing.T) {
	options, err := parseListOptions("2", "50", "  Acme  ", " DISABLED ")
	if err != nil {
		t.Fatalf("parseListOptions returned error: %v", err)
	}
	if options.Page != 2 || options.Limit != 50 || options.Search != "Acme" || options.Status != TeamStatusDisabled {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseListOptionsRejectsInvalidPagination(t *testing.T) {
	tests := []struct {
		name  string
		page  string
		limit string
	}{
		{name: "zero page", page: "0"},
		{name: "non numeric page", page: "abc"},
		{name: "zero limit", limit: "0"},
		{name: "limit above maximum", limit: "101"},
		{name: "non numeric limit", limit: "many"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseListOptions(test.page, test.limit, "", ""); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseListOptionsRejectsInvalidStatus(t *testing.T) {
	if _, err := parseListOptions("", "", "", "pending"); err == nil {
		t.Fatal("expected validation error")
	}
}
