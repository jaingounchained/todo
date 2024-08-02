CREATE TABLE "todos" (
  "id" bigserial PRIMARY KEY,
  "title" varchar(255) NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'incomplete',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "todos" ("status");
