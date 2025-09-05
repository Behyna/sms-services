CREATE TABLE transactions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id bigint NOT NULL,
    tx_type ENUM('topup', 'deduct') NOT NULL,
    amount DECIMAL(18,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user_balances(user_id)
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
