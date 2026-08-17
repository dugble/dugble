package senderid

import "testing"

func TestCreateRequestNormalizeAndValidate(t *testing.T) {
	request := CreateRequest{
		Name:        " DUGBLE ",
		CountryCode: " gh ",
		Purpose:     " Transactional notifications ",
	}.Normalize()

	if request.Name != "DUGBLE" {
		t.Fatalf("Name = %q, want DUGBLE", request.Name)
	}
	if request.CountryCode != "GH" {
		t.Fatalf("CountryCode = %q, want GH", request.CountryCode)
	}
	if request.Purpose != "Transactional notifications" {
		t.Fatalf("Purpose = %q", request.Purpose)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestCreateRequestValidateRejectsInvalidFields(t *testing.T) {
	tests := []CreateRequest{
		{Name: "123", CountryCode: "GH", Purpose: "Transactional"},
		{Name: "DUGBLE", CountryCode: "GHA", Purpose: "Transactional"},
		{Name: "DUGBLE", CountryCode: "GH"},
	}

	for _, request := range tests {
		if err := request.Validate(); err == nil {
			t.Fatalf("Validate() unexpectedly accepted %+v", request)
		}
	}
}
