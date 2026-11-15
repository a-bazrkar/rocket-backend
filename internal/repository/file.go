package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/constant"
	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type File struct {
	db *database.DB
}

func NewFile(db *database.DB) *File {
	return &File{db: db}
}

func (r *File) Create(ctx context.Context, row *model.File) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *File) GetByID(ctx context.Context, id int64) (model.File, error) {
	var row model.File
	err := r.db.GORM(ctx).Where("id = ?", id).First(&row).Error
	return row, err
}

func (r *File) GetByStorageKey(ctx context.Context, storageKey string) (model.File, error) {
	var row model.File
	err := r.db.GORM(ctx).Where("storage_key = ?", storageKey).First(&row).Error
	return row, err
}

func (r *File) ListPending(ctx context.Context) ([]model.File, error) {
	var rows []model.File
	err := r.db.GORM(ctx).
		Where("status = ?", string(constant.FileStatusPending)).
		Order("created_at").
		Find(&rows).Error
	return rows, err
}

func (r *File) UpdateStatus(ctx context.Context, id int64, status string, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.File{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": updatedAt,
	}).Error
}
