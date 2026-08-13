package suppression

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/coffeyvidzro/dugble/server/pkg/pgconv"
)

func (r *Repository) CreateChannel(ctx context.Context, params CreateChannelParams) (ChannelSuppression, error) {
	row, err := r.queries.CreateChannelSuppression(ctx, dbsqlc.CreateChannelSuppressionParams{
		TeamID:            params.TeamID,
		Channel:           params.Channel,
		Address:           params.Address,
		NormalizedAddress: params.NormalizedAddress,
		Reason:            params.Reason,
		Origin:            params.Origin,
		SourceID:          params.SourceID,
	})
	if err != nil {
		return ChannelSuppression{}, fmt.Errorf("create channel suppression: %w", err)
	}
	return channelSuppressionFromSQLC(row), nil
}

func (r *Repository) IsSuppressed(ctx context.Context, teamID uuid.UUID, channel, normalizedAddress string) (bool, error) {
	value, err := r.queries.IsChannelAddressSuppressed(ctx, dbsqlc.IsChannelAddressSuppressedParams{
		TeamID:            teamID,
		Channel:           channel,
		NormalizedAddress: normalizedAddress,
	})
	if err != nil {
		return false, fmt.Errorf("check channel suppression: %w", err)
	}
	return value, nil
}

func (r *Repository) DeleteChannel(ctx context.Context, teamID uuid.UUID, channel, normalizedAddress string) (ChannelSuppression, error) {
	row, err := r.queries.DeleteChannelSuppression(ctx, dbsqlc.DeleteChannelSuppressionParams{
		TeamID:            teamID,
		Channel:           channel,
		NormalizedAddress: normalizedAddress,
	})
	if err != nil {
		return ChannelSuppression{}, fmt.Errorf("delete channel suppression: %w", err)
	}
	return channelSuppressionFromSQLC(row), nil
}

func channelSuppressionFromSQLC(row dbsqlc.ChannelSuppression) ChannelSuppression {
	return ChannelSuppression{
		ID:                row.ID,
		TeamID:            row.TeamID,
		Channel:           row.Channel,
		Address:           row.Address,
		NormalizedAddress: row.NormalizedAddress,
		Reason:            row.Reason,
		Origin:            row.Origin,
		SourceID:          row.SourceID,
		CreatedAt:         pgconv.TimestamptzToTime(row.CreatedAt),
	}
}
