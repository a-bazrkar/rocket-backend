package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/constant"
	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Conversation struct {
	db *database.DB
}

func NewConversation(db *database.DB) *Conversation {
	return &Conversation{db: db}
}

func (r *Conversation) Create(ctx context.Context, row *model.Conversation) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Conversation) GetByID(ctx context.Context, id int64) (model.Conversation, error) {
	var row model.Conversation
	err := r.db.GORM(ctx).Where("id = ?", id).First(&row).Error
	return row, err
}

func (r *Conversation) GetDirect(ctx context.Context, userAID, userBID int64) (model.Conversation, error) {
	var row model.Conversation
	err := r.db.GORM(ctx).
		Where("user_a_id = ? AND user_b_id = ? AND deleted_at IS NULL", userAID, userBID).
		First(&row).Error
	return row, err
}

func (r *Conversation) GetSelf(ctx context.Context, creatorID int64) (model.Conversation, error) {
	var row model.Conversation
	err := r.db.GORM(ctx).
		Where("type = ? AND creator_id = ? AND deleted_at IS NULL", constant.ConversationTypeSelf, creatorID).
		First(&row).Error
	return row, err
}

func (r *Conversation) Update(ctx context.Context, id int64, name, description *string, avatarFileID *int64, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Conversation{}).Where("id = ?", id).Updates(map[string]any{
		"name":           name,
		"description":    description,
		"avatar_file_id": avatarFileID,
		"updated_at":     updatedAt,
	}).Error
}

func (r *Conversation) SoftDelete(ctx context.Context, id, deletedAt, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Conversation{}).Where("id = ?", id).Updates(map[string]any{
		"deleted_at": deletedAt,
		"updated_at": updatedAt,
	}).Error
}
