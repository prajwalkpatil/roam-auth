-- +goose Up
CREATE SCHEMA auth;

CREATE TABLE auth.users (
    id uuid DEFAULT uuidv7() PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE auth.passwords (
    id uuid REFERENCES auth.users(id) PRIMARY KEY,
    hashed_password VARCHAR(255) NOT NULL
);

CREATE TABLE auth.tokens (
    id uuid REFERENCES auth.users(id) NOT NULL,
    refresh_token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE public.users (
    id uuid REFERENCES auth.users(id) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true
);

-- +goose Down
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS auth.passwords;
DROP TABLE IF EXISTS auth.tokens;
DROP TABLE IF EXISTS auth.users;

DROP SCHEMA IF EXISTS auth;