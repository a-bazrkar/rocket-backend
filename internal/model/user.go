package model

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	AvatarFileID *int64
	Status       string
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    *int64
}

func (User) TableName() string {
	return "users"
}
