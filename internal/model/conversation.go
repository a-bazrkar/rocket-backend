package model

type Conversation struct {
	ID           int64
	Type         string
	UserAID      *int64
	UserBID      *int64
	Name         *string
	Description  *string
	AvatarFileID *int64
	ViewHistory  bool
	CreatedBy    int64
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    *int64
}

func (Conversation) TableName() string {
	return "conversations"
}
