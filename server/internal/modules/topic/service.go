package topic

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/platform/audit"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) CreateAPI(ctx context.Context, request CreateRequest) (MutationResponse, error) {
	value, err := s.Create(ctx, request)
	if err != nil {
		return MutationResponse{}, err
	}
	return MutationResponse{Object: ObjectTopic, ID: value.ID}, nil
}

func (s *Service) GetAPI(ctx context.Context, identifier string) (Resource, error) {
	value, err := s.Get(ctx, identifier)
	if err != nil {
		return Resource{}, err
	}
	return resourceFromTopic(value), nil
}

func (s *Service) UpdateAPI(ctx context.Context, identifier string, request UpdateRequest) (MutationResponse, error) {
	value, err := s.Update(ctx, identifier, request)
	if err != nil {
		return MutationResponse{}, err
	}
	return MutationResponse{Object: ObjectTopic, ID: value.ID}, nil
}

func (s *Service) DeleteAPI(ctx context.Context, identifier string) (DeleteResponse, error) {
	value, err := s.Delete(ctx, identifier)
	if err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{Object: ObjectTopic, ID: value.ID, Deleted: true}, nil
}

func resourceFromTopic(value Topic) Resource {
	return Resource{
		Object:              ObjectTopic,
		ID:                  value.ID,
		Name:                value.Name,
		Description:         value.Description,
		DefaultSubscription: value.DefaultSubscription,
		Visibility:          value.Visibility,
		CreatedAt:           value.CreatedAt,
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Topic, error) {
	access, err := requireTenant(ctx, authz.PermissionTopicsWrite)
	if err != nil {
		return Topic{}, err
	}
	validated, err := validateCreate(req)
	if err != nil {
		return Topic{}, err
	}
	value, err := s.repository.Create(ctx, access.Scope.TeamID, validated)
	if err != nil {
		return Topic{}, apperrors.NewInternal("Unable to create topic", err)
	}
	audit.Record(ctx, access, audit.Event{Action: "topic.created", ResourceType: "topic", ResourceID: value.ID})
	return value, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]Topic, error) {
	access, err := requireTenant(ctx, authz.PermissionTopicsRead)
	if err != nil {
		return nil, err
	}
	normalizeListRequest(&req)
	values, err := s.repository.List(ctx, access.Scope.TeamID, req.Limit, req.Offset)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list topics", err)
	}
	return values, nil
}

func (s *Service) Get(ctx context.Context, value string) (Topic, error) {
	access, err := requireTenant(ctx, authz.PermissionTopicsRead)
	if err != nil {
		return Topic{}, err
	}
	id, err := parseID(value)
	if err != nil {
		return Topic{}, err
	}
	result, err := s.repository.Get(ctx, id, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Topic{}, apperrors.NewNotFound("Topic not found")
	}
	if err != nil {
		return Topic{}, apperrors.NewInternal("Unable to get topic", err)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, value string, req UpdateRequest) (Topic, error) {
	access, err := requireTenant(ctx, authz.PermissionTopicsWrite)
	if err != nil {
		return Topic{}, err
	}
	id, err := parseID(value)
	if err != nil {
		return Topic{}, err
	}
	current, err := s.repository.Get(ctx, id, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Topic{}, apperrors.NewNotFound("Topic not found")
	}
	if err != nil {
		return Topic{}, apperrors.NewInternal("Unable to get topic", err)
	}
	name := current.Name
	description := current.Description
	visibility := current.Visibility
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		description = normalizeOptional(*req.Description)
	}
	if req.Visibility != nil {
		visibility = strings.ToLower(strings.TrimSpace(*req.Visibility))
	}
	if err := validateNameDescription(name, description); err != nil {
		return Topic{}, err
	}
	if visibility != "public" && visibility != "private" {
		return Topic{}, apperrors.NewBadRequest("Visibility must be public or private")
	}
	result, err := s.repository.Update(ctx, id, access.Scope.TeamID, name, description, visibility)
	if errors.Is(err, pgx.ErrNoRows) {
		return Topic{}, apperrors.NewNotFound("Topic not found")
	}
	if err != nil {
		return Topic{}, apperrors.NewInternal("Unable to update topic", err)
	}
	audit.Record(ctx, access, audit.Event{Action: "topic.updated", ResourceType: "topic", ResourceID: result.ID})
	return result, nil
}

func (s *Service) Delete(ctx context.Context, value string) (Topic, error) {
	access, err := requireTenant(ctx, authz.PermissionTopicsWrite)
	if err != nil {
		return Topic{}, err
	}
	id, err := parseID(value)
	if err != nil {
		return Topic{}, err
	}
	result, err := s.repository.Delete(ctx, id, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Topic{}, apperrors.NewNotFound("Topic not found")
	}
	if err != nil {
		return Topic{}, apperrors.NewInternal("Unable to delete topic", err)
	}
	audit.Record(ctx, access, audit.Event{Action: "topic.deleted", ResourceType: "topic", ResourceID: result.ID})
	return result, nil
}

func requireTenant(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	access, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return access, nil
}
