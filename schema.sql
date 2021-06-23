CREATE EXTENSION IF NOT EXISTS citext;


DROP Table IF EXISTS users CASCADE;
DROP Table IF EXISTS forums CASCADE;
DROP Table IF EXISTS threads CASCADE;
DROP Table IF EXISTS posts CASCADE;
DROP Table IF EXISTS votes CASCADE;
DROP Table IF EXISTS usersForums CASCADE;


DROP INDEX IF EXISTS index_unique_email;
DROP INDEX IF EXISTS index_unique_nickname;
DROP INDEX IF EXISTS index_unique_slug_forums;
DROP INDEX IF EXISTS index_unique_slug_thread;

DROP INDEX IF EXISTS forum_created_threads;
DROP INDEX IF EXISTS index_post_path;
DROP INDEX IF EXISTS index_post_thread_create_id;
DROP INDEX IF EXISTS index_post_id_thread;

DROP INDEX IF EXISTS index_post_thread_path;
DROP INDEX IF EXISTS votes_thread_nickname;
DROP INDEX IF EXISTS index_users_forums_user;
DROP INDEX IF EXISTS index_users_forums_forum;


CREATE UNLOGGED TABLE IF NOT EXISTS users (
  nickname  CITEXT     PRIMARY KEY,
  fullname  CITEXT,
  about     TEXT,
  email     CITEXT      NOT NULL UNIQUE
);


CREATE INDEX index_unique_email ON users USING HASH (email);

CREATE INDEX index_unique_nickname ON users (nickname);


CREATE UNLOGGED TABLE IF NOT EXISTS forums (
  id        SERIAL      NOT NULL PRIMARY KEY,
  title     VARCHAR     NOT NULL,
  username  CITEXT      NOT NULL REFERENCES users(nickname),
  slug      CITEXT      NOT NULL UNIQUE,
  posts     INTEGER     DEFAULT 0,
  threads   INTEGER     DEFAULT 0
);

CREATE INDEX index_unique_slug_forums ON forums USING HASH (slug);


CREATE UNLOGGED TABLE IF NOT EXISTS threads (
  id        SERIAL                      NOT NULL PRIMARY KEY,
  author    CITEXT                      NOT NULL REFERENCES users(nickname),
  created   TIMESTAMP WITH TIME ZONE    DEFAULT now(),
  forum     CITEXT                      NOT NULL REFERENCES forums(slug),
  message   TEXT                        NOT NULL,
  slug      CITEXT                      UNIQUE,
  title     VARCHAR                     NOT NULL,
  votes     INTEGER                     DEFAULT 0
);


CREATE INDEX index_unique_slug_thread ON threads USING HASH(slug);
CREATE INDEX forum_created_threads on threads (forum, created);

CREATE UNLOGGED TABLE IF NOT EXISTS posts (
  id        BIGSERIAL                   NOT NULL PRIMARY KEY,
  author    CITEXT                      NOT NULL REFERENCES users(nickname) ON DELETE CASCADE,
  created   TIMESTAMP WITH TIME ZONE    DEFAULT now(),
  forum     CITEXT,                     
  isEdited  BOOLEAN                     DEFAULT FALSE,
  message   TEXT                        NOT NULL,
  parent    INTEGER                     DEFAULT 0,
  thread    INTEGER                     NOT NULL,
  path      BIGINT[]
);

CREATE INDEX index_post_path ON posts((path[1]));
CREATE INDEX index_post_thread_create_id ON posts(thread, created, id);
CREATE INDEX index_post_id_thread ON posts(thread, id);
CREATE INDEX index_post_thread_path ON posts(thread, path);

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


CREATE UNLOGGED TABLE IF NOT EXISTS votes (
  id        SERIAL      NOT NULL PRIMARY KEY,
  username  CITEXT      NOT NULL REFERENCES users(nickname),
  voice     INTEGER,
  thread    INTEGER     NOT NULL REFERENCES threads(id),
  UNIQUE(username, thread)
);

create index votes_thread_nickname on votes (thread, username);


CREATE UNLOGGED TABLE IF NOT EXISTS usersForums (
  username         CITEXT     REFERENCES users(nickname) NOT NULL,
  forum            CITEXT     REFERENCES forums(slug) NOT NULL,
  UNIQUE (forum, username)
);

CREATE INDEX index_users_forums_forum ON usersForums USING HASH (forum);
CREATE INDEX index_users_forums ON usersForums (forum, username);


/* ------------------ */

CREATE OR REPLACE FUNCTION add_users_forums()
  RETURNS TRIGGER AS
$add_users_forums$
BEGIN
  INSERT INTO usersForums (username, forum)
  VALUES (new.author, new.forum)
  ON CONFLICT DO NOTHING;
  RETURN new;
END;
$add_users_forums$ LANGUAGE plpgsql;

CREATE TRIGGER add_forum_user_new_post
  AFTER INSERT
  ON posts
  FOR EACH ROW
EXECUTE PROCEDURE add_users_forums();

/* ----------------- */

CREATE OR REPLACE FUNCTION update_forum_posts()
  RETURNS TRIGGER AS
$update_forum_posts$
BEGIN
  UPDATE forums SET posts = posts + 1 WHERE slug = new.forum;
  RETURN new;
END;
$update_forum_posts$ LANGUAGE plpgsql;

CREATE TRIGGER update_forum_posts
  BEFORE INSERT
  ON posts
  FOR EACH ROW
EXECUTE PROCEDURE update_forum_posts();