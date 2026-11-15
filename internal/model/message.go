package model

type Message struct {
	ID                  int64
	ConversationID      int64
	SenderID            int64
	PublicID            *string
	Kind                string
	Content             *string
	SystemEvent         *string
	SystemData          *string
	ReplyToMessageID    *int64
	ForwardedFromUserID *int64
	CreatedAt           int64
	UpdatedAt           int64
	EditedAt            *int64
	DeletedAt           *int64

	// relations
	Conversation      *Conversation
	Sender            *User
	ReplyToMessage    *Message
	ForwardedFromUser *User
	Attachments       []Attachment
	Reactions         []Reaction
}

func (Message) TableName() string {
	return "messages"
}
