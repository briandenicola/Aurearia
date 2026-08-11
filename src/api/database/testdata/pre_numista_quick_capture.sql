CREATE TABLE quick_capture_drafts (
  id integer PRIMARY KEY AUTOINCREMENT,
  user_id integer NOT NULL,
  working_title varchar(200),
  status varchar(20) NOT NULL DEFAULT 'active',
  promoted_coin_id integer,
  promoted_at datetime,
  created_at datetime,
  updated_at datetime
);

INSERT INTO quick_capture_drafts
  (id, user_id, working_title, status, created_at, updated_at)
VALUES
  (1, 1, 'Active pre-feature draft', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO quick_capture_drafts
  (id, user_id, working_title, status, promoted_coin_id, promoted_at, created_at, updated_at)
VALUES
  (2, 1, 'Promoted pre-feature draft', 'promoted', 42, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
