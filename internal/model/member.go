package model

type Member struct {
	ConversationID         int64
	UserID                 int64
	IsHidden               bool
	Role                   string
	NotificationLevel      string
	LeftAt                 *int64
	LeftReason             *string
	IsMuted                bool
	MutedUntil             *int64
	JoinedAt               int64
	LastReadAt             int64
	LastReadMessageID      *int64
	ClearedBeforeMessageID *int64
	UpdatedAt              int64
}

func (Member) TableName() string {
	return "members"
}
