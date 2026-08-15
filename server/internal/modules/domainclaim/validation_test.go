package domainclaim

import "testing"

func TestValidateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     Request
		wantErr bool
	}{
		{name: "valid US claim", req: Request{Name: "Mail.Example.com.", Region: "us-east-1"}},
		{name: "valid EU claim with TLS", req: Request{Name: "mail.example.com", Region: "eu-north-1", TLS: "enforced"}},
		{name: "missing name", req: Request{Region: "us-east-1"}, wantErr: true},
		{name: "invalid name", req: Request{Name: "not a domain", Region: "us-east-1"}, wantErr: true},
		{name: "unsupported region", req: Request{Name: "mail.example.com", Region: "ap-south-1"}, wantErr: true},
		{name: "invalid TLS", req: Request{Name: "mail.example.com", Region: "us-east-1", TLS: "required"}, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			name, region, cfg, err := validateRequest(test.req)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validate request: %v", err)
			}
			if name != "mail.example.com" {
				t.Fatalf("unexpected normalized name: %q", name)
			}
			if region != test.req.Region {
				t.Fatalf("unexpected region: %q", region)
			}
			if cfg.CustomReturnPath != "send" {
				t.Fatalf("unexpected custom return path: %q", cfg.CustomReturnPath)
			}
			if test.req.TLS == "" && cfg.TLS != "opportunistic" {
				t.Fatalf("unexpected default TLS: %q", cfg.TLS)
			}
		})
	}
}
