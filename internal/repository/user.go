package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type User struct {
	db *database.DB
}

func NewUser(db *database.DB) *User {
	return &User{db: db}
}

func (r *User) Create(ctx context.Context, row *model.User) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *User) GetByID(ctx context.Context, id int64) (model.User, error) {
	var row model.User
	err := r.db.GORM(ctx).Where("id = ?", id).First(&row).Error
	return row, err
}

func (r *User) GetByUsername(ctx context.Context, username string) (model.User, error) {
	var row model.User
	err := r.db.GORM(ctx).Where("username = ? AND deleted_at IS NULL", username).First(&row).Error
	return row, err
}

func (r *User) GetByEmail(ctx context.Context, email string) (model.User, error) {
	var row model.User
	err := r.db.GORM(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&row).Error
	return row, err
}

func (r *User) UpdateProfile(ctx context.Context, id int64, displayName string, avatarFileID *int64, status string, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
		"display_name":   displayName,
		"avatar_file_id": avatarFileID,
		"status":         status,
		"updated_at":     updatedAt,
	}).Error
}

func (r *User) UpdatePassword(ctx context.Context, id int64, passwordHash string, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash": passwordHash,
		"updated_at":    updatedAt,
	}).Error
}

func (r *User) SoftDelete(ctx context.Context, id, deletedAt, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
		"deleted_at": deletedAt,
		"updated_at": updatedAt,
	}).Error
}
