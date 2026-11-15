package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Auth struct {
	db *database.DB
}

func NewAuth(db *database.DB) *Auth {
	return &Auth{db: db}
}

func (r *Auth) Create(ctx context.Context, row *model.Token) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Auth) GetByHash(ctx context.Context, tokenHash string) (model.Token, error) {
	var row model.Token
	err := r.db.GORM(ctx).Where("token_hash = ?", tokenHash).First(&row).Error
	return row, err
}

func (r *Auth) ListByUser(ctx context.Context, userID int64) ([]model.Token, error) {
	var rows []model.Token
	err := r.db.GORM(ctx).Where("user_id = ?", userID).Order("id").Find(&rows).Error
	return rows, err
}

func (r *Auth) Revoke(ctx context.Context, id, revokedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Token{}).Where("id = ?", id).Update("revoked_at", revokedAt).Error
}
