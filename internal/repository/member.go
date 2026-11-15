package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Member struct {
	db *database.DB
}

func NewMember(db *database.DB) *Member {
	return &Member{db: db}
}

func (r *Member) Create(ctx context.Context, row *model.Member) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Member) Get(ctx context.Context, conversationID, userID int64) (model.Member, error) {
	var row model.Member
	err := r.db.GORM(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&row).Error
	return row, err
}

func (r *Member) ListByConversation(ctx context.Context, conversationID int64) ([]model.Member, error) {
	var rows []model.Member
	err := r.db.GORM(ctx).Where("conversation_id = ?", conversationID).Find(&rows).Error
	return rows, err
}

func (r *Member) ListActiveByUser(ctx context.Context, userID int64) ([]model.Member, error) {
	var rows []model.Member
	err := r.db.GORM(ctx).Where("user_id = ? AND left_at IS NULL", userID).Find(&rows).Error
	return rows, err
}

func (r *Member) SetHidden(ctx context.Context, conversationID, userID int64, hidden bool, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"is_hidden":  hidden,
		"updated_at": updatedAt,
	})
}

func (r *Member) SetRole(ctx context.Context, conversationID, userID int64, role string, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"role":       role,
		"updated_at": updatedAt,
	})
}

func (r *Member) SetNotification(ctx context.Context, conversationID, userID int64, level string, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"notification_level": level,
		"updated_at":         updatedAt,
	})
}

func (r *Member) SetLeft(ctx context.Context, conversationID, userID int64, leftAt *int64, reason *string, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"left_at":     leftAt,
		"left_reason": reason,
		"updated_at":  updatedAt,
	})
}

func (r *Member) SetMute(ctx context.Context, conversationID, userID int64, muted bool, mutedUntil *int64, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"is_muted":    muted,
		"muted_until": mutedUntil,
		"updated_at":  updatedAt,
	})
}

func (r *Member) SetLastRead(ctx context.Context, conversationID, userID, readAt int64, messageID *int64, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"last_read_at":         readAt,
		"last_read_message_id": messageID,
		"updated_at":           updatedAt,
	})
}

func (r *Member) SetClearedBefore(ctx context.Context, conversationID, userID int64, messageID *int64, updatedAt int64) error {
	return r.update(ctx, conversationID, userID, map[string]any{
		"cleared_before_message_id": messageID,
		"updated_at":                updatedAt,
	})
}

func (r *Member) update(ctx context.Context, conversationID, userID int64, values map[string]any) error {
	return r.db.GORM(ctx).Model(&model.Member{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(values).Error
}
