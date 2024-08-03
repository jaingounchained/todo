ALTER TABLE todos
ADD COLUMN file_count INTEGER DEFAULT 0 NOT NULL;

CREATE TABLE "attachments" (
    id bigserial PRIMARY KEY,
    todo_id bigint NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    storage_filename VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (todo_id) REFERENCES todos (id) ON DELETE CASCADE
);

