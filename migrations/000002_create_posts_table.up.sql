CREATE TABLE IF NOT EXISTS posts (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    images text[] NOT NULL,
    content text NOT NULL,
    views bigint NOT NULL DEFAULT 0,
    version integer NOT NULL DEFAULT 1
)