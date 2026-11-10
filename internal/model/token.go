package model

type Token struct {
	ID         int64
	UserID     int64
	TokenHash  string
	DeviceName *string
	CreatedAt  int64
	ExpiresAt  int64
	RevokedAt  *int64
}

func (Token) TableName() string {
	return "tokens"
}
