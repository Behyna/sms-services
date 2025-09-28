CREATE TABLE transactions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id CHAR(11) NOT NULL,
    idempotency_key VARCHAR(36) NOT NULL ,
    tx_type ENUM('INCREASE', 'DECREASE') NOT NULL,
    amount INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user_balances(user_id)
);

CREATE UNIQUE INDEX idx_transactions_idempotency_txType ON transactions(idempotency_key, tx_type);
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
