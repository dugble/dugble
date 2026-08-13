package smscampaign

import (
	"strings"
	"testing"
)

const validID = "11111111-1111-4111-8111-111111111111"

func TestValidateCreateStoresBodyDirectly(t *testing.T) {
	t.Parallel()
	body := "Hello Ada — your order is ready."
	_, _, request, err := validateCreate(CreateRequest{
		Name: " August customers ", SegmentID: validID, SenderID: validID, Body: body,
	})
	if err != nil {
		t.Fatalf("validateCreate() error = %v", err)
	}
	if request.Name != "August customers" {
		t.Fatalf("name = %q", request.Name)
	}
	if request.Body != body {
		t.Fatalf("body changed: got %q", request.Body)
	}
}

func TestValidateCreateRejectsOversizedBody(t *testing.T) {
	t.Parallel()
	_, _, _, err := validateCreate(CreateRequest{
		Name: "Campaign", SegmentID: validID, SenderID: validID,
		Body: strings.Repeat("a", maxBodyCharacters+1),
	})
	if err == nil {
		t.Fatal("validateCreate() accepted an oversized SMS body")
	}
}
