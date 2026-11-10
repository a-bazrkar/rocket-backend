package model

type Friend struct {
	RequesterID int64
	AddresseeID int64
	Status      string
	CreatedAt   int64
	UpdatedAt   int64
}

func (Friend) TableName() string {
	return "friends"
}
