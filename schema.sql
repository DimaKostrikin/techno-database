CREATE EXTENSION citext;


DROP Table IF EXISTS users CASCADE;
DROP Table IF EXISTS forums CASCADE;
DROP Table IF EXISTS threads CASCADE;
DROP Table IF EXISTS posts CASCADE;
DROP Table IF EXISTS votes CASCADE;
DROP Table IF EXISTS usersForums CASCADE;


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
DROP TRIGGER IF EXISTS insert_thread_votes ON votes;
DROP TRIGGER IF EXISTS update_thread_votes ON votes;


CREATE UNLOGGED TABLE IF NOT EXISTS users (
  nickname  CITEXT     PRIMARY KEY,
  fullname  CITEXT,
  about     TEXT,
  email     CITEXT      NOT NULL UNIQUE
);


CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueEmail ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS uniqueUpNickname ON users(UPPER(nickname));
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueNickname ON users(nickname);
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueNicknameLow ON users(LOWER(nickname collate "ucs_basic"));


CREATE UNLOGGED TABLE IF NOT EXISTS forums (
  id        SERIAL      NOT NULL PRIMARY KEY,
  title     VARCHAR     NOT NULL,
  username  CITEXT      NOT NULL REFERENCES users(nickname),
  slug      CITEXT      NOT NULL UNIQUE,
  posts     INTEGER     DEFAULT 0,
  threads   INTEGER     DEFAULT 0
);


CREATE INDEX IF NOT EXISTS indexForumsUser ON forums(username);
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueSlugForums ON forums(slug);


CREATE UNLOGGED TABLE IF NOT EXISTS threads (
  id        SERIAL                      NOT NULL PRIMARY KEY,
  author    CITEXT                      NOT NULL REFERENCES users(nickname),
  created   TIMESTAMP WITH TIME ZONE    DEFAULT current_timestamp,
  forum     CITEXT                      NOT NULL REFERENCES forums(slug),
  message   TEXT                        NOT NULL,
  slug      CITEXT                      UNIQUE,
  title     VARCHAR                     NOT NULL,
  votes     INTEGER                     DEFAULT 0
);


CREATE INDEX IF NOT EXISTS indexThreadUser ON threads(author);
CREATE INDEX IF NOT EXISTS indexThreadForum ON threads(forum);
CREATE UNIQUE INDEX IF NOT EXISTS indexUniqueSlugThread ON threads(slug);


CREATE UNLOGGED TABLE IF NOT EXISTS posts (
  id        BIGSERIAL                   NOT NULL PRIMARY KEY,
  author    CITEXT                      NOT NULL REFERENCES users(nickname),
  created   TIMESTAMP WITH TIME ZONE    DEFAULT current_timestamp,
  forum     VARCHAR,
  isEdited  BOOLEAN                     DEFAULT FALSE,
  message   TEXT                        NOT NULL,
  parent    INTEGER                     DEFAULT 0,
  thread    INTEGER                     NOT NULL REFERENCES threads(id),
  path      BIGINT[]
);

CREATE OR REPLACE FUNCTION set_post_path()
    RETURNS TRIGGER AS
$set_post_path$
DECLARE
    parent_thread BIGINT;
    parent_path   BIGINT[];
BEGIN
    IF (new.parent = 0) THEN
        new.path := new.path || new.id;
    ELSE
        SELECT thread, path
        FROM posts p
        WHERE p.thread = new.thread
          AND p.id = new.parent
        INTO parent_thread , parent_path;
        IF parent_thread != new.thread OR NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '00404';
        END IF;
        new.path := parent_path || new.id;
    END IF;
    RETURN new;
END;
$set_post_path$ LANGUAGE plpgsql;

CREATE TRIGGER set_post_path
    BEFORE INSERT
    ON posts
    FOR EACH ROW
EXECUTE PROCEDURE set_post_path();


CREATE INDEX IF NOT EXISTS indexPostAuthor ON posts(author);
CREATE INDEX IF NOT EXISTS indexPostForum ON posts(forum);
CREATE INDEX IF NOT EXISTS indexPostThread ON posts(thread);
CREATE INDEX IF NOT EXISTS indexPostCreated ON posts(created);
CREATE INDEX IF NOT EXISTS indexPostPath ON posts((path[1]));
CREATE INDEX IF NOT EXISTS indexPostThreadCreateId ON posts(thread, created, id);
CREATE INDEX IF NOT EXISTS indexPostParentThreadId ON posts(parent, thread, id);
CREATE INDEX IF NOT EXISTS indexPostIdThread ON posts(id, thread);
CREATE INDEX IF NOT EXISTS indexPostThreadPath ON posts(thread, path);


CREATE UNLOGGED TABLE IF NOT EXISTS votes (
  id        SERIAL      NOT NULL PRIMARY KEY,
  username  CITEXT      NOT NULL REFERENCES users(nickname),
  voice     INTEGER,
  thread    INTEGER     NOT NULL REFERENCES threads(id),
  UNIQUE(username, thread)
);


CREATE UNLOGGED TABLE IF NOT EXISTS usersForums (
  username         CITEXT     REFERENCES users(nickname) NOT NULL,
  forum            CITEXT     REFERENCES forums(slug) NOT NULL,
  UNIQUE (forum, username)
);

CREATE OR REPLACE FUNCTION insert_thread_votes()
    RETURNS TRIGGER AS
$insert_thread_votes$
BEGIN
    IF new.voice > 0 THEN
        UPDATE threads SET votes = (votes + 1)
        WHERE id = new.thread;
    ELSE
        UPDATE threads SET votes = (votes - 1)
        WHERE id = new.thread;
    END IF;
    RETURN new;
END;
$insert_thread_votes$ language plpgsql;

CREATE TRIGGER insert_thread_votes
    BEFORE INSERT
    ON votes
    FOR EACH ROW
EXECUTE PROCEDURE insert_thread_votes();



CREATE OR REPLACE FUNCTION update_thread_votes()
    RETURNS TRIGGER AS
$update_thread_votes$
BEGIN
    IF new.voice > 0 THEN
        UPDATE threads
        SET votes = (votes + 2)
        WHERE threads.id = new.thread;
    else
        UPDATE threads
        SET votes = (votes - 2)
        WHERE threads.id = new.thread;
    END IF;
    RETURN new;
END;
$update_thread_votes$ LANGUAGE plpgsql;

CREATE TRIGGER update_thread_votes
    BEFORE UPDATE
    ON votes
    FOR EACH ROW
EXECUTE PROCEDURE update_thread_votes();


CREATE INDEX IF NOT EXISTS indexUsersForumsUser ON usersForums (username);
CREATE INDEX IF NOT EXISTS indexUsersForumsForum ON usersForums (forum);