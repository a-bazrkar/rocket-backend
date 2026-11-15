package service

import (
	"context"

	"github.com/a-bazrkar/rocket-backend/internal/model"
	"github.com/a-bazrkar/rocket-backend/internal/packages/database"
)

type UserRepository interface {
	Create(ctx context.Context, row *model.User) error
	GetByID(ctx context.Context, id int64) (model.User, error)
	GetByUsername(ctx context.Context, username string) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	UpdateProfile(ctx context.Context, id int64, displayName string, avatarFileID *int64, status string, updatedAt int64) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string, updatedAt int64) error
	SoftDelete(ctx context.Context, id, deletedAt, updatedAt int64) error
}

type AuthRepository interface {
	Create(ctx context.Context, row *model.Token) error
	GetByHash(ctx context.Context, tokenHash string) (model.Token, error)
	ListByUser(ctx context.Context, userID int64) ([]model.Token, error)
	Revoke(ctx context.Context, id, revokedAt int64) error
}

type FriendRepository interface {
	Create(ctx context.Context, row *model.Friend) error
	Get(ctx context.Context, requesterID, addresseeID int64) (model.Friend, error)
	ListByRequester(ctx context.Context, requesterID int64) ([]model.Friend, error)
	ListByAddressee(ctx context.Context, addresseeID int64) ([]model.Friend, error)
	Accept(ctx context.Context, requesterID, addresseeID, updatedAt int64) error
	Delete(ctx context.Context, requesterID, addresseeID int64) error
}

type BlockRepository interface {
	Create(ctx context.Context, row *model.Block) error
	Get(ctx context.Context, blockerID, blockedID int64) (model.Block, error)
	ListByBlocker(ctx context.Context, blockerID int64) ([]model.Block, error)
	ListByBlocked(ctx context.Context, blockedID int64) ([]model.Block, error)
	Delete(ctx context.Context, blockerID, blockedID int64) error
}

type FileRepository interface {
	Create(ctx context.Context, row *model.File) error
	GetByID(ctx context.Context, id int64) (model.File, error)
	GetByStorageKey(ctx context.Context, storageKey string) (model.File, error)
	ListPending(ctx context.Context) ([]model.File, error)
	UpdateStatus(ctx context.Context, id int64, status string, updatedAt int64) error
}

type ConversationRepository interface {
	Create(ctx context.Context, row *model.Conversation) error
	GetByID(ctx context.Context, id int64) (model.Conversation, error)
	GetDirect(ctx context.Context, userAID, userBID int64) (model.Conversation, error)
	GetSelf(ctx context.Context, creatorID int64) (model.Conversation, error)
	Update(ctx context.Context, id int64, name, description *string, avatarFileID *int64, updatedAt int64) error
	SoftDelete(ctx context.Context, id, deletedAt, updatedAt int64) error
}

type MemberRepository interface {
	Create(ctx context.Context, row *model.Member) error
	Get(ctx context.Context, conversationID, userID int64) (model.Member, error)
	ListByConversation(ctx context.Context, conversationID int64) ([]model.Member, error)
	ListActiveByUser(ctx context.Context, userID int64) ([]model.Member, error)
	SetHidden(ctx context.Context, conversationID, userID int64, hidden bool, updatedAt int64) error
	SetRole(ctx context.Context, conversationID, userID int64, role string, updatedAt int64) error
	SetNotification(ctx context.Context, conversationID, userID int64, level string, updatedAt int64) error
	SetLeft(ctx context.Context, conversationID, userID int64, leftAt *int64, reason *string, updatedAt int64) error
	SetMute(ctx context.Context, conversationID, userID int64, muted bool, mutedUntil *int64, updatedAt int64) error
	SetLastRead(ctx context.Context, conversationID, userID, readAt int64, messageID *int64, updatedAt int64) error
	SetClearedBefore(ctx context.Context, conversationID, userID int64, messageID *int64, updatedAt int64) error
}

type MessageRepository interface {
	Create(ctx context.Context, row *model.Message) error
	GetByID(ctx context.Context, id int64) (model.Message, error)
	GetByPublicID(ctx context.Context, senderID int64, publicID string) (model.Message, error)
	ListBefore(ctx context.Context, conversationID, beforeID int64, limit int) ([]model.Message, error)
	ListUpdatedSince(ctx context.Context, conversationID, updatedAt, id int64, limit int) ([]model.Message, error)
	UpdateContent(ctx context.Context, id int64, content *string, editedAt, updatedAt int64) error
	SoftDelete(ctx context.Context, id, deletedAt, updatedAt int64) error
}

type AttachmentRepository interface {
	Create(ctx context.Context, row *model.Attachment) error
	ListByMessage(ctx context.Context, messageID int64) ([]model.Attachment, error)
	Delete(ctx context.Context, messageID, fileID int64) error
}

type ReactionRepository interface {
	Create(ctx context.Context, row *model.Reaction) error
	ListByMessage(ctx context.Context, messageID int64) ([]model.Reaction, error)
	Delete(ctx context.Context, messageID, userID int64, emoji string) error
}

type InviteRepository interface {
	Create(ctx context.Context, row *model.Invite) error
	GetByCode(ctx context.Context, code string) (model.Invite, error)
	ListByConversation(ctx context.Context, conversationID int64) ([]model.Invite, error)
	IncrementUse(ctx context.Context, code string) error
	Revoke(ctx context.Context, code string, revokedAt int64) error
}

type Repository struct {
	UserRepository
	AuthRepository
	FriendRepository
	BlockRepository
	FileRepository
	ConversationRepository
	MemberRepository
	MessageRepository
	AttachmentRepository
	ReactionRepository
	InviteRepository
}

type Service struct {
	transactor *database.Transactor
	repository Repository
}

func NewService(transactor *database.Transactor, repository Repository) *Service {
	return &Service{transactor: transactor, repository: repository}
}
