package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPrintAtlasMigrationChecksum(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Clean(filepath.Join(wd, "../../../migrations"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	cumulative := sha256.New()
	type item struct{ name, hash string }
	items := make([]item, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = cumulative.Write([]byte(name))
		_, _ = cumulative.Write(data)
		items = append(items, item{name: name, hash: base64.StdEncoding.EncodeToString(cumulative.Sum(nil))})
	}

	top := sha256.New()
	var body strings.Builder
	for _, item := range items {
		_, _ = top.Write([]byte(item.name))
		_, _ = top.Write([]byte(item.hash))
		fmt.Fprintf(&body, "%s h1:%s\n", item.name, item.hash)
	}
	result := fmt.Sprintf("h1:%s\n%s", base64.StdEncoding.EncodeToString(top.Sum(nil)), body.String())
	t.Fatalf("EXPECTED_ATLAS_SUM_BEGIN\n%sEXPECTED_ATLAS_SUM_END", result)
}
