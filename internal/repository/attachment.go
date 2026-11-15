package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Attachment struct {
	db *database.DB
}

func NewAttachment(db *database.DB) *Attachment {
	return &Attachment{db: db}
}

func (r *Attachment) Create(ctx context.Context, row *model.Attachment) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Attachment) ListByMessage(ctx context.Context, messageID int64) ([]model.Attachment, error) {
	var rows []model.Attachment
	err := r.db.GORM(ctx).Where("message_id = ?", messageID).Order("position").Find(&rows).Error
	return rows, err
}

func (r *Attachment) Delete(ctx context.Context, messageID, fileID int64) error {
	return r.db.GORM(ctx).
		Where("message_id = ? AND file_id = ?", messageID, fileID).
		Delete(&model.Attachment{}).Error
}
