-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP default current_timestamp,
    is_active BOOLEAN DEFAULT true,
    encrypted_password VARCHAR(255)
);

-- +goose Down
DROP TABLE users;