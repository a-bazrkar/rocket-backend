package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Invite struct {
	db *database.DB
}

func NewInvite(db *database.DB) *Invite {
	return &Invite{db: db}
}

func (r *Invite) Create(ctx context.Context, row *model.Invite) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Invite) GetByCode(ctx context.Context, code string) (model.Invite, error) {
	var row model.Invite
	err := r.db.GORM(ctx).Where("code = ?", code).First(&row).Error
	return row, err
}

func (r *Invite) ListByConversation(ctx context.Context, conversationID int64) ([]model.Invite, error) {
	var rows []model.Invite
	err := r.db.GORM(ctx).Where("conversation_id = ?", conversationID).Find(&rows).Error
	return rows, err
}

func (r *Invite) IncrementUse(ctx context.Context, code string) error {
	return r.db.GORM(ctx).Model(&model.Invite{}).
		Where("code = ?", code).
		Update("use_count", gorm.Expr("use_count + 1")).Error
}

func (r *Invite) Revoke(ctx context.Context, code string, revokedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Invite{}).
		Where("code = ?", code).
		Update("revoked_at", revokedAt).Error
}
