package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Message struct {
	db *database.DB
}

func NewMessage(db *database.DB) *Message {
	return &Message{db: db}
}

func (r *Message) Create(ctx context.Context, row *model.Message) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Message) GetByID(ctx context.Context, id int64) (model.Message, error) {
	var row model.Message
	err := r.db.GORM(ctx).Where("id = ?", id).First(&row).Error
	return row, err
}

func (r *Message) GetByPublicID(ctx context.Context, senderID int64, publicID string) (model.Message, error) {
	var row model.Message
	err := r.db.GORM(ctx).
		Where("sender_id = ? AND public_id = ?", senderID, publicID).
		First(&row).Error
	return row, err
}

func (r *Message) ListBefore(ctx context.Context, conversationID, beforeID int64, limit int) ([]model.Message, error) {
	var rows []model.Message
	q := r.db.GORM(ctx).Where("conversation_id = ?", conversationID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	err := q.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Message) ListUpdatedSince(ctx context.Context, conversationID, updatedAt, id int64, limit int) ([]model.Message, error) {
	var rows []model.Message
	err := r.db.GORM(ctx).
		Where(
			"conversation_id = ? AND (updated_at > ? OR (updated_at = ? AND id > ?))",
			conversationID, updatedAt, updatedAt, id,
		).
		Order("updated_at, id").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *Message) UpdateContent(ctx context.Context, id int64, content *string, editedAt, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Message{}).Where("id = ?", id).Updates(map[string]any{
		"content":    content,
		"edited_at":  editedAt,
		"updated_at": updatedAt,
	}).Error
}

func (r *Message) SoftDelete(ctx context.Context, id, deletedAt, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Message{}).Where("id = ?", id).Updates(map[string]any{
		"deleted_at": deletedAt,
		"updated_at": updatedAt,
	}).Error
}
