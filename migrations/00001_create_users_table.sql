-- +goose Up
CREATE SCHEMA auth;

CREATE TABLE auth.users (
    id uuid DEFAULT uuidv7() PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE auth.passwords (
    id uuid REFERENCES auth.users(id) PRIMARY KEY,
    encrypted_password VARCHAR(255) NOT NULL
);

CREATE TABLE public.users (
    id uuid REFERENCES auth.users(id) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true
);

-- +goose Down
DROP TABLE public.users;
DROP TABLE auth.passwords;
DROP TABLE auth.users;

DROP SCHEMA auth;