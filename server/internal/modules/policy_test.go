package modules_test

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
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read modules directory: %v", err)
	}

	var invalid []string
	for _, module := range entries {
		if !module.IsDir() {
			continue
		}
		files, err := os.ReadDir(module.Name())
		if err != nil {
			t.Fatalf("read module %q: %v", module.Name(), err)
		}
		for _, file := range files {
			name := file.Name()
			if file.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			if _, ok := allowed[name]; !ok {
				invalid = append(invalid, filepath.Join(module.Name(), name))
			}
		}
	}

	if len(invalid) != 0 {
		sort.Strings(invalid)
		t.Fatalf("production module files must follow the vertical module policy; move: %s", strings.Join(invalid, ", "))
	}
}
