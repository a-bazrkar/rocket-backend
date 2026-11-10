package model

type Block struct {
	BlockerID int64
	BlockedID int64
	BlockedAt int64
}

func (Block) TableName() string {
	return "blocks"
}
