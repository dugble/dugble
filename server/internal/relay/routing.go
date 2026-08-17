package relay

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrNoRoutesConfigured = errors.New("no routes configured")
	ErrInvalidRoute       = errors.New("invalid route")
	ErrDuplicateRoute     = errors.New("duplicate route")
	ErrDuplicatePriority  = errors.New("duplicate route priority")
)

// Route declares provider preference for a country. Lower priority values are
// attempted first.
type Route struct {
	Provider    string
	CountryCode string
	Priority    int
	Enabled     bool
}

// RouteTable stores normalized, enabled provider routes keyed by country.
type RouteTable struct {
	byCountry map[string][]Route
}

func NewRouteTable(routes []Route) (*RouteTable, error) {
	if len(routes) == 0 {
		return nil, ErrNoRoutesConfigured
	}

	byCountry := make(map[string][]Route)
	providers := make(map[string]struct{}, len(routes))
	priorities := make(map[string]struct{}, len(routes))

	for _, route := range routes {
		route.Provider = normalizeProviderName(route.Provider)
		route.CountryCode = normalizeCountryCode(route.CountryCode)
		if route.Provider == "" || !isCountryCode(route.CountryCode) || route.Priority < 1 {
			return nil, fmt.Errorf("%w: provider=%q country=%q priority=%d", ErrInvalidRoute, route.Provider, route.CountryCode, route.Priority)
		}

		providerKey := route.CountryCode + "\x00" + route.Provider
		if _, exists := providers[providerKey]; exists {
			return nil, fmt.Errorf("%w: provider %q country %q", ErrDuplicateRoute, route.Provider, route.CountryCode)
		}
		providers[providerKey] = struct{}{}

		priorityKey := fmt.Sprintf("%s\x00%d", route.CountryCode, route.Priority)
		if _, exists := priorities[priorityKey]; exists {
			return nil, fmt.Errorf("%w: country %q priority %d", ErrDuplicatePriority, route.CountryCode, route.Priority)
		}
		priorities[priorityKey] = struct{}{}

		if route.Enabled {
			byCountry[route.CountryCode] = append(byCountry[route.CountryCode], route)
		}
	}

	for country := range byCountry {
		sort.SliceStable(byCountry[country], func(i, j int) bool {
			return byCountry[country][i].Priority < byCountry[country][j].Priority
		})
	}

	return &RouteTable{byCountry: byCountry}, nil
}

// ProviderNames returns enabled providers in priority order for a country.
func (t *RouteTable) ProviderNames(countryCode string) []string {
	if t == nil {
		return nil
	}
	countryCode = normalizeCountryCode(countryCode)
	routes := t.byCountry[countryCode]
	if len(routes) == 0 {
		return nil
	}

	result := make([]string, 0, len(routes))
	for _, route := range routes {
		result = append(result, route.Provider)
	}
	return result
}

func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeCountryCode(countryCode string) string {
	return strings.ToUpper(strings.TrimSpace(countryCode))
}

func isCountryCode(countryCode string) bool {
	if len(countryCode) != 2 {
		return false
	}
	for _, r := range countryCode {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
