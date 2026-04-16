PRAGMA foreign_keys = ON;

CREATE TABLE tickets (
	id INTEGER PRIMARY KEY,
	subject TEXT NOT NULL,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	requester_email TEXT NOT NULL,
	metadata JSONB,
	created_at DATETIME,
	updated_at DATETIME
);

CREATE TABLE ticket_comments (
	id INTEGER PRIMARY KEY,
	ticket_id INTEGER NOT NULL REFERENCES tickets(id),
	author_email TEXT NOT NULL,
	body TEXT NOT NULL,
	created_at DATETIME
);

CREATE INDEX idx_ticket_comments_ticket_id ON ticket_comments(ticket_id);
