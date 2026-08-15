package domain

import "testing"

func TestDomainNotificationStatus(t *testing.T) {
	tests := []struct {
		name     string
		previous SenderDomain
		updated  SenderDomain
		want     string
	}{
		{name: "verified", previous: SenderDomain{Status: StatusPending}, updated: SenderDomain{Status: StatusVerified}, want: StatusVerified},
		{name: "failed", previous: SenderDomain{Status: StatusPending}, updated: SenderDomain{Status: StatusFailed}, want: StatusFailed},
		{name: "degraded", previous: SenderDomain{Status: StatusVerified, HealthStatus: HealthStatusHealthy}, updated: SenderDomain{Status: StatusVerified, HealthStatus: HealthStatusDegraded}, want: HealthStatusDegraded},
		{name: "verified unchanged", previous: SenderDomain{Status: StatusVerified}, updated: SenderDomain{Status: StatusVerified}},
		{name: "degraded unchanged", previous: SenderDomain{HealthStatus: HealthStatusDegraded}, updated: SenderDomain{HealthStatus: HealthStatusDegraded}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NotificationStatus(test.previous, test.updated); got != test.want {
				t.Fatalf("NotificationStatus() = %q, want %q", got, test.want)
			}
		})
	}
}
