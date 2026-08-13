package campaign

import "testing"

func TestRenderBodyPersonalizesSnapshotAndProperties(t *testing.T) {
	t.Parallel()
	body, err := renderBody("Hi {{first_name}}, your code is {{code}}.", map[string]any{"first_name": "Ada", "properties": map[string]any{"code": "1234"}})
	if err != nil {
		t.Fatalf("renderBody() error = %v", err)
	}
	if body != "Hi Ada, your code is 1234." {
		t.Fatalf("body = %q", body)
	}
}

func TestRenderBodyRejectsMissingVariable(t *testing.T) {
	t.Parallel()
	if _, err := renderBody("Hi {{first_name}}", map[string]any{}); err == nil {
		t.Fatal("renderBody() accepted missing variable")
	}
}

func TestRenderBodyTreatsSnapshottedNullAsEmpty(t *testing.T) {
	t.Parallel()
	body, err := renderBody("Hi {{first_name}}", map[string]any{"first_name": nil})
	if err != nil {
		t.Fatalf("renderBody() error = %v", err)
	}
	if body != "Hi " {
		t.Fatalf("body = %q", body)
	}
}
