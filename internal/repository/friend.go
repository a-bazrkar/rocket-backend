package repository

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/constant"
	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
	"gorm.io/gorm/clause"
)

type Friend struct {
	db *database.DB
}

func NewFriend(db *database.DB) *Friend {
	return &Friend{db: db}
}

func (r *Friend) Create(ctx context.Context, row *model.Friend) error {
	return r.db.GORM(ctx).Omit(clause.Associations).Create(row).Error
}

func (r *Friend) Get(ctx context.Context, requesterID, addresseeID int64) (model.Friend, error) {
	var row model.Friend
	err := r.db.GORM(ctx).
		Where("requester_id = ? AND addressee_id = ?", requesterID, addresseeID).
		First(&row).Error
	return row, err
}

func (r *Friend) ListByRequester(ctx context.Context, requesterID int64) ([]model.Friend, error) {
	var rows []model.Friend
	err := r.db.GORM(ctx).Where("requester_id = ?", requesterID).Find(&rows).Error
	return rows, err
}

func (r *Friend) ListByAddressee(ctx context.Context, addresseeID int64) ([]model.Friend, error) {
	var rows []model.Friend
	err := r.db.GORM(ctx).Where("addressee_id = ?", addresseeID).Find(&rows).Error
	return rows, err
}

func (r *Friend) Accept(ctx context.Context, requesterID, addresseeID, updatedAt int64) error {
	return r.db.GORM(ctx).Model(&model.Friend{}).
		Where("requester_id = ? AND addressee_id = ?", requesterID, addresseeID).
		Updates(map[string]any{
			"status":     string(constant.FriendStatusAccepted),
			"updated_at": updatedAt,
		}).Error
}

func (r *Friend) Delete(ctx context.Context, requesterID, addresseeID int64) error {
	return r.db.GORM(ctx).
		Where("requester_id = ? AND addressee_id = ?", requesterID, addresseeID).
		Delete(&model.Friend{}).Error
}
