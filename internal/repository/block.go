package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Block struct {
	db *database.DB
}

func NewBlock(db *database.DB) *Block {
	return &Block{db: db}
}

func (r *Block) Create(ctx context.Context, row *model.Block) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Block) Get(ctx context.Context, blockerID, blockedID int64) (model.Block, error) {
	var row model.Block
	err := r.db.GORM(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		First(&row).Error
	return row, err
}

func (r *Block) ListByBlocker(ctx context.Context, blockerID int64) ([]model.Block, error) {
	var rows []model.Block
	err := r.db.GORM(ctx).Where("blocker_id = ?", blockerID).Find(&rows).Error
	return rows, err
}

func (r *Block) ListByBlocked(ctx context.Context, blockedID int64) ([]model.Block, error) {
	var rows []model.Block
	err := r.db.GORM(ctx).Where("blocked_id = ?", blockedID).Find(&rows).Error
	return rows, err
}

func (r *Block) Delete(ctx context.Context, blockerID, blockedID int64) error {
	return r.db.GORM(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&model.Block{}).Error
}
