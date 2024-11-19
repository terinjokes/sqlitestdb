CREATE TABLE users (
       id INTEGER NOT NULL PRIMARY KEY,
       name TEXT
);

CREATE TABLE blog_posts (
       id INTEGER NOT NULL PRIMARY KEY,
       title TEXT,
       body TEXT,
       author_id INTEGER,
       CONSTRAINT author_fk FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE NO ACTION
);
