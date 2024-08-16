CREATE TABLE IF NOT EXISTS author (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    name text NOT NULL,
    email text NOT NULL,
    presentation text,
    avatar text NOT NULL,
    birth text NOT NULL,
    location text NOT NULL,
    status_activity text NOT NULL,
    formations text[],
    experiences text[],
    tags text[],
    cv_file text NOT NULL,
    version integer NOT NULL DEFAULT 1
)