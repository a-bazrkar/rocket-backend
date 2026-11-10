package constant

const (
	UsernameMinLength = 3
	UsernameMaxLength = 32
	EmojiMinLength    = 1
	EmojiMaxLength    = 32
)

type UserStatus string

const (
	UserStatusDefault UserStatus = "default"
	UserStatusDND     UserStatus = "dnd"
	UserStatusOffline UserStatus = "offline"
)

type FriendStatus string

const (
	FriendStatusPending  FriendStatus = "pending"
	FriendStatusAccepted FriendStatus = "accepted"
)

type FileStatus string

const (
	FileStatusPending  FileStatus = "pending"
	FileStatusVerified FileStatus = "verified"
)

type ConversationType string

const (
	ConversationTypeSelf   ConversationType = "self"
	ConversationTypeDirect ConversationType = "direct"
	ConversationTypeGroup  ConversationType = "group"
)

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

type NotificationLevel string

const (
	NotificationLevelAll      NotificationLevel = "all"
	NotificationLevelMentions NotificationLevel = "mentions"
	NotificationLevelNone     NotificationLevel = "none"
)

type LeftReason string

const (
	LeftReasonLeft   LeftReason = "left"
	LeftReasonKicked LeftReason = "kicked"
	LeftReasonBanned LeftReason = "banned"
)

type MessageKind string

const (
	MessageKindText   MessageKind = "text"
	MessageKindSystem MessageKind = "system"
)
