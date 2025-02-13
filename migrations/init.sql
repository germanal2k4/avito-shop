-- +goose Up
CREATE TABLE IF NOT EXISTS users (
                                     id SERIAL PRIMARY KEY,
                                     username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    coins INTEGER DEFAULT 1000
    );

CREATE TABLE IF NOT EXISTS inventories (
                                           id SERIAL PRIMARY KEY,
                                           user_id INTEGER REFERENCES users(id),
    item_type VARCHAR(50) NOT NULL,
    quantity INTEGER DEFAULT 0
    );

CREATE TABLE IF NOT EXISTS coin_transactions (
                                                 id SERIAL PRIMARY KEY,
                                                 user_id INTEGER REFERENCES users(id),
    transaction_type VARCHAR(10) NOT NULL, -- 'sent' или 'received'
    counterparty VARCHAR(50) NOT NULL,
    amount INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- +goose Down
DROP TABLE IF EXISTS coin_transactions;
DROP TABLE IF EXISTS inventories;
DROP TABLE IF EXISTS users;
