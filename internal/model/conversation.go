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
	CreatorID    int64
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    *int64

	// relations
	UserA      *User
	UserB      *User
	AvatarFile *File
	Creator    *User
	Members    []Member
}

func (Conversation) TableName() string {
	return "conversations"
}
