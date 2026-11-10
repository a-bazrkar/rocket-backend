package model

type File struct {
	ID         int64
	UploaderID int64
	StorageKey string
	FileName   string
	MimeType   string
	SizeBytes  int64
	SHA256     []byte
	Width      *int64
	Height     *int64
	DurationMs *int64
	Status     string
	CreatedAt  int64
	UpdatedAt  int64
}

func (File) TableName() string {
	return "files"
}
