
-- ---------------------------- USERS ---------------------------------------

CREATE TABLE users (
    id                   INTEGER PRIMARY KEY,

    -- user credentials
    username             TEXT NOT NULL COLLATE NOCASE CHECK (length(username) BETWEEN 3 AND 32),
    email                TEXT NOT NULL COLLATE NOCASE,
    password_hash        TEXT NOT NULL,

    -- user profile
    display_name         TEXT NOT NULL,
    avatar_file_id       INTEGER REFERENCES files(id),
    last_seen_at         INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'default' CHECK (status IN ('default', 'dnd', 'offline')),

    -- visibility
    avatar_visibility    TEXT NOT NULL DEFAULT 'everyone' CHECK (avatar_visibility IN ('everyone', 'friends', 'nobody')),
    last_seen_visibility TEXT NOT NULL DEFAULT 'everyone' CHECK (last_seen_visibility IN ('everyone', 'friends', 'nobody')),

    -- common
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL,
    deleted_at           INTEGER

) STRICT;

CREATE UNIQUE INDEX users_username_uq ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_email_uq    ON users (email)    WHERE deleted_at IS NULL;
CREATE INDEX users_avatar_idx ON users (avatar_file_id) WHERE avatar_file_id IS NOT NULL;


-- ---------------------------- TOKENS --------------------------------------

CREATE TABLE tokens (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    token_hash  TEXT NOT NULL UNIQUE,
    device_name TEXT,
    created_at  INTEGER NOT NULL,
    expires_at  INTEGER NOT NULL,
    revoked_at  INTEGER

) STRICT;

CREATE INDEX tokens_user_idx    ON tokens (user_id);
CREATE INDEX tokens_expires_idx ON tokens (expires_at);


-- ---------------------------- FRIENDSHIPS --------------------------------

CREATE TABLE friendships (
    requester_id INTEGER NOT NULL REFERENCES users(id),
    addressee_id INTEGER NOT NULL REFERENCES users(id),
    status       TEXT NOT NULL CHECK (status IN ('pending', 'accepted')),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,

    PRIMARY KEY (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id)

) STRICT, WITHOUT ROWID;

CREATE UNIQUE INDEX friendships_pair_uq
    ON friendships (min(requester_id, addressee_id), max(requester_id, addressee_id));
CREATE INDEX friendships_addressee_idx ON friendships (addressee_id);


-- ---------------------------- BLOCKS --------------------------------------

CREATE TABLE blocks (
    blocker_id INTEGER NOT NULL REFERENCES users(id),
    blocked_id INTEGER NOT NULL REFERENCES users(id),
    blocked_at INTEGER NOT NULL,

    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)

) STRICT, WITHOUT ROWID;

CREATE INDEX blocks_blocked_idx ON blocks (blocked_id);


-- ---------------------------- FILES ---------------------------------------

CREATE TABLE files (
    id           INTEGER PRIMARY KEY,
    uploader_id  INTEGER NOT NULL REFERENCES users(id),
    storage_key  TEXT NOT NULL UNIQUE,

    -- file metadata
    file_name    TEXT NOT NULL,
    mime_type    TEXT NOT NULL,
    size_bytes   INTEGER NOT NULL CHECK (size_bytes > 0),
    sha256       BLOB NOT NULL CHECK (length(sha256) = 32),

    -- image / video
    width        INTEGER,
    height       INTEGER,

    -- video / audio
    duration_ms  INTEGER,

    -- common
    status       TEXT NOT NULL CHECK (status IN ('pending', 'verified')),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
) STRICT;

CREATE INDEX files_pending_idx ON files (created_at) WHERE status = 'pending';


-- ---------------------------- CONVERSATIONS --------------------------------

CREATE TABLE conversations (
    id              INTEGER PRIMARY KEY,
    type            TEXT NOT NULL CHECK (type IN ('self', 'direct', 'group')),

    -- direct conversation (user_a_id < user_b_id)
    user_a_id       INTEGER REFERENCES users(id),
    user_b_id       INTEGER REFERENCES users(id),

    -- group conversation
    name            TEXT,
    description     TEXT,
    avatar_file_id  INTEGER REFERENCES files(id),
    view_history    INTEGER NOT NULL CHECK (view_history IN (0, 1)),

    -- common
    created_by      INTEGER NOT NULL REFERENCES users(id),
    last_message_at INTEGER,

    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    deleted_at      INTEGER,

    CHECK (
        (type = 'group'  AND user_a_id IS NULL AND user_b_id IS NULL AND name IS NOT NULL) OR
        (type = 'direct' AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL AND user_a_id < user_b_id) OR
        (type = 'self'   AND user_a_id IS NULL AND user_b_id IS NULL)
    )
    
) STRICT;

CREATE UNIQUE INDEX conversations_pair_uq
    ON conversations (user_a_id, user_b_id) WHERE type = 'direct' AND deleted_at IS NULL;
CREATE UNIQUE INDEX conversations_self_uq
    ON conversations (created_by) WHERE type = 'self' AND deleted_at IS NULL;
CREATE INDEX conversations_avatar_idx
    ON conversations (avatar_file_id) WHERE avatar_file_id IS NOT NULL;


-- ---------------------------- MEMBERS -------------------------------------

CREATE TABLE members (
    conversation_id           INTEGER NOT NULL REFERENCES conversations(id),
    user_id                   INTEGER NOT NULL REFERENCES users(id),

    -- direct conversation
    is_hidden                 INTEGER NOT NULL CHECK (is_hidden IN (0, 1)),

    -- group conversation
    role                      TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    notification_level        TEXT NOT NULL CHECK (notification_level IN ('all', 'mentions', 'none')),
    left_at                   INTEGER,
    left_reason               TEXT CHECK (left_reason IN ('left', 'kicked', 'banned')),
    is_muted                  INTEGER NOT NULL CHECK (is_muted IN (0, 1)),
    muted_until               INTEGER,

    -- common
    joined_at                 INTEGER NOT NULL,
    last_read_at              INTEGER NOT NULL,
    last_read_message_id      INTEGER REFERENCES messages(id),
    cleared_before_message_id INTEGER REFERENCES messages(id),

    updated_at                INTEGER NOT NULL,

    PRIMARY KEY (conversation_id, user_id),
    CHECK ((left_at IS NULL) = (left_reason IS NULL)),
    CHECK (is_muted = 1 OR muted_until IS NULL)

) STRICT, WITHOUT ROWID;

CREATE INDEX members_user_idx ON members (user_id) WHERE left_at IS NULL;


-- ---------------------------- MESSAGES ------------------------------------

CREATE TABLE messages (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id        INTEGER NOT NULL REFERENCES conversations(id),
    sender_id              INTEGER NOT NULL REFERENCES users(id),
    public_id              TEXT,

    -- message kind
    kind                   TEXT NOT NULL CHECK (kind IN ('text', 'system')),

    -- text message
    content                TEXT,

    -- system message
    system_event           TEXT,
    system_data            TEXT,

    -- message references
    reply_to_message_id    INTEGER REFERENCES messages(id),
    forwarded_from_user_id INTEGER REFERENCES users(id),

    -- common
    created_at             INTEGER NOT NULL,
    updated_at             INTEGER NOT NULL,
    edited_at              INTEGER,
    deleted_at             INTEGER,

    CHECK ((kind = 'system') = (system_event IS NOT NULL))

) STRICT;

CREATE INDEX messages_conv_id_idx      ON messages (conversation_id, id);          -- history paging, unread counts
CREATE INDEX messages_conv_updated_idx ON messages (conversation_id, updated_at);  -- client sync
CREATE UNIQUE INDEX messages_public_id_uq  ON messages (sender_id, public_id) WHERE public_id IS NOT NULL;


-- ---------------------------- ATTACHMENTS ---------------------------------

CREATE TABLE attachments (
    message_id INTEGER NOT NULL REFERENCES messages(id),
    file_id    INTEGER NOT NULL REFERENCES files(id),
    position   INTEGER NOT NULL,

    PRIMARY KEY (message_id, file_id)

) STRICT, WITHOUT ROWID;

CREATE INDEX attachments_file_idx ON attachments (file_id);


-- ---------------------------- REACTIONS -----------------------------------

CREATE TABLE reactions (
    message_id INTEGER NOT NULL REFERENCES messages(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    emoji      TEXT NOT NULL CHECK (length(emoji) BETWEEN 1 AND 32),
    created_at INTEGER NOT NULL,

    PRIMARY KEY (message_id, user_id, emoji)

) STRICT, WITHOUT ROWID;


-- ---------------------------- INVITES -------------------------------------

CREATE TABLE invites (
    code            TEXT PRIMARY KEY,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id),
    created_by      INTEGER NOT NULL REFERENCES users(id),
    max_uses        INTEGER CHECK (max_uses IS NULL OR max_uses > 0),
    use_count       INTEGER NOT NULL CHECK (use_count >= 0),
    expires_at      INTEGER,
    revoked_at      INTEGER,
    created_at      INTEGER NOT NULL,

    CHECK (max_uses IS NULL OR use_count <= max_uses)

) STRICT, WITHOUT ROWID;

CREATE INDEX invites_conversation_idx ON invites (conversation_id);
