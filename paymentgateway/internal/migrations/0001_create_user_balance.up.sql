CREATE TABLE user_balances (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id CHAR(11) UNIQUE NOT NULL,
    balance INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
               ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_balance_user_id ON user_balances(user_id);
