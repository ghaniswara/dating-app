CREATE TABLE IF NOT EXISTS swipe_transactions (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    to_id BIGINT NOT NULL REFERENCES users(id),
    date DATE NOT NULL,
    action SMALLINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    is_matched BOOLEAN NOT NULL
);

CREATE INDEX idx_user_id ON swipe_transactions (user_id);
CREATE INDEX idx_to_id ON swipe_transactions (to_id);
CREATE INDEX idx_date ON swipe_transactions (date);
