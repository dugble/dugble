package users

import (
	"context"
	"testing"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type fakeRepository struct {
	rows   []Row
	filter Filter
}

func (r *fakeRepository) List(_ context.Context, filter Filter) ([]Row, error) {
	r.filter = filter
	return r.rows, nil
}
func (r *fakeRepository) Detail(context.Context, string) (Detail, error) { return Detail{}, nil }

func TestListAppliesPagination(t *testing.T) {
	repository := &fakeRepository{rows: []Row{{ID: "1"}, {ID: "2"}, {ID: "3"}}}
	page, err := NewService(repository).List(context.Background(), Filter{Query: " user ", Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repository.filter.Query != "user" || repository.filter.Limit != 3 || repository.filter.Offset != 4 {
		t.Fatalf("repository filter = %#v", repository.filter)
	}
	if len(page.Data) != 2 || !page.HasMore || page.Limit != 2 || page.Offset != 4 {
		t.Fatalf("page = %#v", page)
	}
}

func TestDetailRejectsInvalidUserID(t *testing.T) {
	_, err := NewService(&fakeRepository{}).Detail(context.Background(), "not-a-uuid")
	if !apperrors.IsCode(err, apperrors.CodeBadRequest) {
		t.Fatalf("Detail() error = %v", err)
	}
}
