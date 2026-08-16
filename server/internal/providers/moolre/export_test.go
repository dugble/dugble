package moolre

// NewTestProvider constructs a Moolre provider with a test-only endpoint.
func NewTestProvider(config Config, baseURL string) (*Provider, error) {
	return newProvider(config, baseURL)
}
