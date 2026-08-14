package domainreconciliation

import (
	"testing"

	domainmodule "github.com/dugble/dugble/server/internal/modules/domain"
)

func TestDomainNotificationStatus(t *testing.T) {
	tests := []struct {
		name     string
		previous domainmodule.SenderDomain
		updated  domainmodule.SenderDomain
		want     string
	}{
		{name: "verified", previous: domainmodule.SenderDomain{Status: domainmodule.StatusPending}, updated: domainmodule.SenderDomain{Status: domainmodule.StatusVerified}, want: domainmodule.StatusVerified},
		{name: "failed", previous: domainmodule.SenderDomain{Status: domainmodule.StatusPending}, updated: domainmodule.SenderDomain{Status: domainmodule.StatusFailed}, want: domainmodule.StatusFailed},
		{name: "degraded", previous: domainmodule.SenderDomain{Status: domainmodule.StatusVerified, HealthStatus: domainmodule.HealthStatusHealthy}, updated: domainmodule.SenderDomain{Status: domainmodule.StatusVerified, HealthStatus: domainmodule.HealthStatusDegraded}, want: domainmodule.HealthStatusDegraded},
		{name: "verified unchanged", previous: domainmodule.SenderDomain{Status: domainmodule.StatusVerified}, updated: domainmodule.SenderDomain{Status: domainmodule.StatusVerified}},
		{name: "degraded unchanged", previous: domainmodule.SenderDomain{HealthStatus: domainmodule.HealthStatusDegraded}, updated: domainmodule.SenderDomain{HealthStatus: domainmodule.HealthStatusDegraded}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := domainmodule.NotificationStatus(test.previous, test.updated); got != test.want {
				t.Fatalf("NotificationStatus() = %q, want %q", got, test.want)
			}
		})
	}
}
