package model

type Block struct {
	BlockerID int64
	BlockedID int64
	BlockedAt int64

	// relations
	Blocker *User
	Blocked *User
}

func (Block) TableName() string {
	return "blocks"
}
