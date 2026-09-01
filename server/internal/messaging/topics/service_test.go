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

func TestNormalizeListRequestPreservesMaximumPageLookahead(t *testing.T) {
	t.Parallel()

	request := ListRequest{Limit: maxListLimit + listLookahead, Offset: 10}
	normalizeListRequest(&request)

	if request.Limit != 101 {
		t.Fatalf("limit = %d, want 101", request.Limit)
	}
	if request.Offset != 10 {
		t.Fatalf("offset = %d, want 10", request.Offset)
	}
}

func TestNormalizeListRequestRejectsLimitBeyondLookahead(t *testing.T) {
	t.Parallel()

	request := ListRequest{Limit: maxListLimit + listLookahead + 1}
	normalizeListRequest(&request)

	if request.Limit != defaultListLimit {
		t.Fatalf("limit = %d, want %d", request.Limit, defaultListLimit)
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
