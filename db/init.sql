CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    balance NUMERIC(15, 2) NOT NULL DEFAULT 0.0
);

INSERT INTO clients(name, balance) VALUES ('Jhon Doe', 10000.00);
INSERT INTO clients(name, balance) VALUES ('Judy Doe', 10000.00);

CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    from_account_id INT NOT NULL,
    to_account_id INT NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'pending',
    FOREIGN KEY (from_account_id) REFERENCES clients(id),
    FOREIGN KEY (to_account_id) REFERENCES clients(id)
);

CREATE INDEX IF NOT EXISTS idx_from_account_id ON transactions (from_account_id);
CREATE INDEX IF NOT EXISTS idx_to_account_id ON transactions (to_account_id);