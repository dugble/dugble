package teams

import (
	"context"
	"testing"

	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
)

type fakeRepository struct {
	rows          []Row
	filter        Filter
	updatedID     string
	updatedStatus string
	detail        Detail
}

func (r *fakeRepository) List(_ context.Context, filter Filter) ([]Row, error) {
	r.filter = filter
	return r.rows, nil
}
func (r *fakeRepository) Detail(context.Context, string) (Detail, error) { return r.detail, nil }
func (r *fakeRepository) UpdateStatus(_ context.Context, id, status string) error {
	r.updatedID, r.updatedStatus = id, status
	return nil
}

func TestListNormalizesFiltersAndPaginates(t *testing.T) {
	repository := &fakeRepository{rows: []Row{{ID: "1"}, {ID: "2"}, {ID: "3"}}}
	page, err := NewService(repository).List(context.Background(), Filter{Query: " team ", Status: " ACTIVE ", Limit: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repository.filter.Query != "team" || repository.filter.Status != "active" || repository.filter.Limit != 3 {
		t.Fatalf("filter = %#v", repository.filter)
	}
	if len(page.Data) != 2 || !page.HasMore {
		t.Fatalf("page = %#v", page)
	}
}

func TestUpdateStatusValidatesAndReturnsDetail(t *testing.T) {
	const id = "0d063004-b23b-4c09-a387-2e491012e701"
	repository := &fakeRepository{detail: Detail{Team: Row{ID: id, Status: "disabled"}}}
	detail, err := NewService(repository).UpdateStatus(context.Background(), id, StatusRequest{Status: " DISABLED ", Reason: " abuse investigation "})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if repository.updatedID != id || repository.updatedStatus != "disabled" {
		t.Fatalf("update = %q, %q", repository.updatedID, repository.updatedStatus)
	}
	if detail.Team.Status != "disabled" {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestUpdateStatusRejectsInvalidRequests(t *testing.T) {
	const id = "0d063004-b23b-4c09-a387-2e491012e701"
	for _, request := range []StatusRequest{{Status: "archived", Reason: "bad"}, {Status: "active"}} {
		_, err := NewService(&fakeRepository{}).UpdateStatus(context.Background(), id, request)
		if !apperrors.IsCode(err, apperrors.CodeBadRequest) {
			t.Fatalf("UpdateStatus(%#v) error = %v", request, err)
		}
	}
}
