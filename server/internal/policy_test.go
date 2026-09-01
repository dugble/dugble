package internal_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestVerticalModuleFileNames keeps production code in the small, predictable
// set of files that define a vertical module. Tests may use descriptive names.
func TestVerticalModuleFileNames(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"model.go": {}, "repository.go": {}, "service.go": {},
		"validation.go": {}, "handler.go": {}, "routes.go": {},
		"consumer.go": {}, "publisher.go": {}, "jobs.go": {},
	}
	// Business areas contain vertical modules at their first level. Shared
	// contracts and infrastructure below them may use responsibility-specific
	// filenames. Keep the nested vertical modules in scope explicitly.
	modules := []string{
		"messaging/domains/claims", "messaging/email/tenants",
		"modules/audit/events", "modules/webhooks", "runtime/server/health",
	}
	for _, area := range []string{"identity", "tenancy", "messaging", "audience", "campaigns"} {
		entries, err := os.ReadDir(area)
		if err != nil {
			t.Fatalf("read business area %q: %v", area, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() || (area == "messaging" && entry.Name() == "unsubscribe") {
				continue
			}
			modules = append(modules, filepath.Join(area, entry.Name()))
		}
	}

	var invalid []string
	for _, module := range modules {
		files, err := os.ReadDir(module)
		if err != nil {
			t.Fatalf("read module %q: %v", module, err)
		}
		for _, file := range files {
			name := file.Name()
			if file.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			if _, ok := allowed[name]; !ok {
				invalid = append(invalid, filepath.Join(module, name))
			}
		}
	}

	if len(invalid) != 0 {
		sort.Strings(invalid)
		t.Fatalf("production module files must follow the vertical module policy; move: %s", strings.Join(invalid, ", "))
	}
}
