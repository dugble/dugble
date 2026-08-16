package sendexa

// NewTestProvider constructs a Sendexa provider with a test-only endpoint.
func NewTestProvider(config Config, baseURL string) (*Provider, error) {
	return newProvider(config, baseURL)
}
