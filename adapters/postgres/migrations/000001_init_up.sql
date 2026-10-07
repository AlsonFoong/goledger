-- CREATE TYPE account_type AS ENUM('ASSET', 'LIABILITY', 'REVENUE', 'EXPENSES', 'EQUITY');
-- CREATE TYPE entry_direction AS ENUM('DEBIT', 'CREDIT');

CREATE TABLE accounts (
    id BIGINT PRIMARY KEY,
    -- type account_type NOT NULL,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE transactions (
    id BIGINT PRIMARY KEY,
    sequence_number BIGSERIAL UNIQUE NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE TABLE ledger_entries (
    id SERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    transaction_sequence_number BIGINT NOT NULL REFERENCES transactions(sequence_number) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    -- entry_direction entry_direction NOT NULL,
    entry_direction VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0)
);

CREATE TABLE snapshots (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    total_amount BIGINT NOT NULL,
    last_processed_transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE
);

CREATE INDEX idx_entries_account_id ON ledger_entries(account_id);
CREATE INDEX idx_transactions_idempotency_key ON transactions(idempotency_key);
CREATE INDEX idx_transaction_sequence_number ON ledger_entries(account_id, transaction_sequence_number);