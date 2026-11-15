package model

type Invite struct {
	Code           string
	ConversationID int64
	CreatorID      int64
	MaxUses        *int64
	UseCount       int64
	ExpiresAt      *int64
	RevokedAt      *int64
	CreatedAt      int64

	// relations
	Conversation *Conversation
	Creator      *User
}

func (Invite) TableName() string {
	return "invites"
}
