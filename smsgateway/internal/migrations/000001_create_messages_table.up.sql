CREATE TABLE messages (
    id               BIGINT AUTO_INCREMENT PRIMARY KEY,
    client_message_id VARCHAR(255) NOT NULL,
    from_msisdn      VARCHAR(255) NOT NULL,
    to_msisdn        VARCHAR(255) NOT NULL,
    text             TEXT NOT NULL,
    status           ENUM('QUEUED','SENDING','SUBMITTED','FAILED_TEMP','FAILED_PERM','REFUNDED') NOT NULL DEFAULT 'QUEUED',
    attempt_count    INT NOT NULL DEFAULT 0,
    last_attempt_at  TIMESTAMP NULL,
    provider         VARCHAR(255),
    provider_msg_id  VARCHAR(255),
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY idx_messages_client_msg_from (client_message_id, from_msisdn)
);
