package topic

import (
	"encoding/json"
	"testing"
)

func TestTopicResponseContracts(t *testing.T) {
	mutation, err := json.Marshal(MutationResponse{Object: ObjectTopic, ID: "topic-id"})
	if err != nil {
		t.Fatal(err)
	}
	if string(mutation) != `{"object":"topic","id":"topic-id"}` {
		t.Fatalf("unexpected mutation response: %s", mutation)
	}
	deleted, err := json.Marshal(DeleteResponse{Object: ObjectTopic, ID: "topic-id", Deleted: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(deleted) != `{"object":"topic","id":"topic-id","deleted":true}` {
		t.Fatalf("unexpected delete response: %s", deleted)
	}
}

func TestCreateTopicDefaultsPrivate(t *testing.T) {
	request, err := validateCreate(CreateRequest{Name: "Newsletter", DefaultSubscription: "opt_in"})
	if err != nil {
		t.Fatal(err)
	}
	if request.Visibility != "private" {
		t.Fatalf("visibility = %q, want private", request.Visibility)
	}
}
