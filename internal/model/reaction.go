package model

type Reaction struct {
	MessageID int64
	UserID    int64
	Emoji     string
	CreatedAt int64
}

func (Reaction) TableName() string {
	return "reactions"
}
