package messagetemplate

import (
	"strings"
	"testing"
)

func TestWithPreviewText(t *testing.T) {
	t.Parallel()

	preview := `New <features> & fixes`
	body := `<p>Hello</p>`
	got := withPreviewText(body, &preview)
	if !strings.Contains(got, `data-dugble-preheader='true'`) {
		t.Fatalf("withPreviewText() missing preheader marker: %q", got)
	}
	if !strings.Contains(got, `New &lt;features&gt; &amp; fixes`) {
		t.Fatalf("withPreviewText() did not escape preview text: %q", got)
	}
	if !strings.HasSuffix(got, body) {
		t.Fatalf("withPreviewText() body = %q, want suffix %q", got, body)
	}
}

func TestIsBroadcastTemplate(t *testing.T) {
	t.Parallel()

	internal := "__broadcast_123"
	public := "welcome"
	if !isBroadcastTemplate(Template{Alias: &internal}) {
		t.Fatal("internal broadcast alias was not detected")
	}
	if isBroadcastTemplate(Template{Alias: &public}) || isBroadcastTemplate(Template{}) {
		t.Fatal("public template was detected as broadcast-owned")
	}
}
