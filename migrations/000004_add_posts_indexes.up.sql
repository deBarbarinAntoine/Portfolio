CREATE INDEX IF NOT EXISTS posts_title_idx ON posts USING GIN (to_tsvector('simple', title));

CREATE INDEX IF NOT EXISTS posts_images_idx ON posts USING GIN (images);

CREATE INDEX IF NOT EXISTS posts_content_idx ON posts USING GIN (to_tsvector('simple', content));