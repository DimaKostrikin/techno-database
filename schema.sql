CREATE EXTENSION citext;


DROP Table IF EXISTS users CASCADE;
DROP Table IF EXISTS forums CASCADE;
DROP Table IF EXISTS threads CASCADE;
DROP Table IF EXISTS posts CASCADE;
DROP Table IF EXISTS votes CASCADE;


DROP INDEX IF EXISTS indexUniqueEmail;
DROP INDEX IF EXISTS indexUniqueNickname;
DROP INDEX IF EXISTS indexUniqueNicknameLow;

DROP INDEX IF EXISTS indexForumsUser;
DROP INDEX IF EXISTS indexUniqueSlugForums;

DROP INDEX IF EXISTS indexThreadUser;
DROP INDEX IF EXISTS indexThreadForum;
DROP INDEX IF EXISTS indexUniqueSlugThread;

DROP INDEX IF EXISTS indexPostAuthor;
DROP INDEX IF EXISTS indexPostForum;
DROP INDEX IF EXISTS indexPostThread;
DROP INDEX IF EXISTS indexPostCreated;
DROP INDEX IF EXISTS indexPostThreadCreateId;
DROP INDEX IF EXISTS indexPostParentThreadId;
DROP INDEX IF EXISTS indexPostIdThread;
DROP INDEX IF EXISTS indexPostThreadPath;
DROP INDEX IF EXISTS indexPostPath;

DROP INDEX IF EXISTS indexUsersForumsUser;
DROP INDEX IF EXISTS indexUsersForumsForum;
DROP INDEX IF EXISTS indexUsersForumsUserLow;


CREATE TABLE IF NOT EXISTS users (
  nickname  VARCHAR     PRIMARY KEY,
  fullname  VARCHAR,
  about     TEXT,
  email     CITEXT      NOT NULL UNIQUE
);


CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueEmail ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS uniqueUpNickname ON users(UPPER(nickname));
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueNickname ON users(nickname);
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueNicknameLow ON users(LOWER(nickname collate "ucs_basic"));


CREATE TABLE IF NOT EXISTS forums (
  id        SERIAL      NOT NULL PRIMARY KEY,
  title     VARCHAR     NOT NULL,
  username  VARCHAR     NOT NULL REFERENCES users(nickname),
  slug      CITEXT      NOT NULL UNIQUE,
  posts     INTEGER     DEFAULT 0,
  threads   INTEGER     DEFAULT 0
);


CREATE INDEX IF NOT EXISTS indexForumsUser ON forums(username);
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueSlugForums ON forums(slug);


CREATE TABLE IF NOT EXISTS threads (
  id        SERIAL                      NOT NULL PRIMARY KEY,
  author    VARCHAR                     NOT NULL REFERENCES users(nickname),
  created   TIMESTAMP WITH TIME ZONE    DEFAULT current_timestamp,
  forum     INTEGER                     NOT NULL REFERENCES forums(id),
  message   TEXT                        NOT NULL,
  slug      CITEXT                      UNIQUE,
  title     VARCHAR                     NOT NULL,
  votes     INTEGER                     DEFAULT 0
);


CREATE INDEX IF NOT EXISTS indexThreadUser ON threads(author);
CREATE INDEX IF NOT EXISTS indexThreadForum ON threads(forum);
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueSlugThread ON threads(slug);


CREATE TABLE IF NOT EXISTS posts (
  id        SERIAL                      NOT NULL PRIMARY KEY,
  author    VARCHAR                     NOT NULL REFERENCES users(nickname),
  created   TIMESTAMP WITH TIME ZONE    DEFAULT current_timestamp,
  forum     VARCHAR,
  isEdited  BOOLEAN                     DEFAULT FALSE,
  message   TEXT                        NOT NULL,
  parent    INTEGER                     DEFAULT 0,
  thread    INTEGER                     NOT NULL REFERENCES threads(id),
  path      BIGINT                      ARRAY
);


CREATE INDEX IF NOT EXISTS indexPostAuthor ON posts(author);
CREATE INDEX IF NOT EXISTS indexPostForum ON posts(forum);
CREATE INDEX IF NOT EXISTS indexPostThread ON posts(thread);
CREATE INDEX IF NOT EXISTS indexPostCreated ON posts(created);
CREATE INDEX IF NOT EXISTS indexPostPath ON posts((path[1]));
CREATE INDEX IF NOT EXISTS indexPostThreadCreateId ON posts(thread, created, id);
CREATE INDEX IF NOT EXISTS indexPostParentThreadId ON posts(parent, thread, id);
CREATE INDEX IF NOT EXISTS indexPostIdThread ON posts(id, thread);
CREATE INDEX IF NOT EXISTS indexPostThreadPath ON posts(thread, path);


CREATE TABLE IF NOT EXISTS votes (
  id        SERIAL      NOT NULL PRIMARY KEY,
  username  VARCHAR     NOT NULL REFERENCES users(nickname),
  voice     INTEGER,
  thread    INTEGER     NOT NULL REFERENCES threads(id),
  UNIQUE(username, thread)
);


CREATE TABLE IF NOT EXISTS usersForums (
  user_nickname     VARCHAR     REFERENCES users(nickname) NOT NULL,
  forum_id          INTEGER     REFERENCES forums(id) NOT NULL
);


CREATE INDEX IF NOT EXISTS indexUsersForumsUser ON usersForums (user_nickname);
CREATE INDEX IF NOT EXISTS indexUsersForumsForum ON usersForums (forum_id);
CREATE INDEX IF NOT EXISTS indexUsersForumsUserLow on usersForums (lower(user_nickname) COLLATE "ucs_basic");