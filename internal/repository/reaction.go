package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Reaction struct {
	db *database.DB
}

func NewReaction(db *database.DB) *Reaction {
	return &Reaction{db: db}
}

func (r *Reaction) Create(ctx context.Context, row *model.Reaction) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Reaction) ListByMessage(ctx context.Context, messageID int64) ([]model.Reaction, error) {
	var rows []model.Reaction
	err := r.db.GORM(ctx).Where("message_id = ?", messageID).Find(&rows).Error
	return rows, err
}

func (r *Reaction) Delete(ctx context.Context, messageID, userID int64, emoji string) error {
	return r.db.GORM(ctx).
		Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&model.Reaction{}).Error
}
