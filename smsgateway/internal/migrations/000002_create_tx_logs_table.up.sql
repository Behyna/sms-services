CREATE TABLE tx_logs (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    message_id    BIGINT NOT NULL,
    from_msisdn   VARCHAR(255) NOT NULL,
    amount        INT NOT NULL DEFAULT 1,
    state         ENUM('PENDING','SUCCESS','REFUNDED','FAILED') NOT NULL,
    published     BOOLEAN NOT NULL DEFAULT false,
    published_at  TIMESTAMP NULL,
    last_error    TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id)
);