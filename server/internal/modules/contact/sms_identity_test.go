package contact

import "testing"

func TestValidateCreateNormalizesSMSIdentity(t *testing.T) {
	t.Parallel()

	source := "import"
	phone := "+233 20 000 0000"
	request, err := validateCreate(CreateRequest{
		Email: "ada@example.com", Phone: &phone,
		SMSConsentStatus: SMSConsentOptedIn, SMSConsentSource: &source,
	})
	if err != nil {
		t.Fatalf("validateCreate() error = %v", err)
	}
	if request.NormalizedPhone == nil || *request.NormalizedPhone != "+233200000000" {
		t.Fatalf("normalized phone = %v", request.NormalizedPhone)
	}
	if request.Phone == nil || *request.Phone != "+233 20 000 0000" {
		t.Fatalf("display phone = %v", request.Phone)
	}
	if request.PhoneCountry == nil || *request.PhoneCountry != "GH" {
		t.Fatalf("phone country = %v", request.PhoneCountry)
	}
}

func TestValidateCreateRequiresConsentSource(t *testing.T) {
	t.Parallel()

	_, err := validateCreate(CreateRequest{Email: "ada@example.com", SMSConsentStatus: SMSConsentOptedOut})
	if err == nil {
		t.Fatal("validateCreate() accepted explicit consent without source")
	}
}

func TestValidateCreateRejectsNationalPhone(t *testing.T) {
	t.Parallel()

	phone := "0200000000"
	_, err := validateCreate(CreateRequest{Email: "ada@example.com", Phone: &phone})
	if err == nil {
		t.Fatal("validateCreate() accepted phone without international country code")
	}
}

func TestValidateCreateRejectsUnknownConsentSource(t *testing.T) {
	t.Parallel()

	source := "spreadsheet"
	_, err := validateCreate(CreateRequest{
		Email: "ada@example.com", SMSConsentStatus: SMSConsentOptedIn, SMSConsentSource: &source,
	})
	if err == nil {
		t.Fatal("validateCreate() accepted an unknown SMS consent source")
	}
}
