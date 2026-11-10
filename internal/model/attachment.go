package model

type Attachment struct {
	MessageID int64
	FileID    int64
	Position  int64
}

func (Attachment) TableName() string {
	return "attachments"
}
