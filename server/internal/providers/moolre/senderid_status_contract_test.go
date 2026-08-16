package moolre_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/providers/moolre"
)

func TestCheckSenderIDStatusUsesHTTPContract(t *testing.T) {
	cases := []struct {
		name     string
		senderID string
		approval string
		want     provider.SenderIDStatus
	}{
		