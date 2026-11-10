package model

type Invite struct {
	Code           string
	ConversationID int64
	CreatedBy      int64
	MaxUses        *int64
	UseCount       int64
	ExpiresAt      *int64
	RevokedAt      *int64
	CreatedAt      int64
}

func (Invite) TableName() string {
	return "invites"
}
