-- 迁移到复合主键 (conversation_id, seq)
-- 警告：这个脚本会重建 messages 和 message_sequences 表，请先备份数据！

-- 1. 备份现有数据
CREATE TABLE messages_backup AS SELECT * FROM messages;
CREATE TABLE message_sequences_backup AS SELECT * FROM message_sequences;

-- 2. 删除旧表
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS message_sequences CASCADE;

-- 3. 创建新的 messages 表（使用复合主键）
CREATE TABLE messages (
    conversation_id VARCHAR(64) NOT NULL,
    seq BIGINT NOT NULL,
    client_msg_id VARCHAR(64) NOT NULL,
    sender_id VARCHAR(64) NOT NULL,
    receiver_id VARCHAR(64),
    group_id VARCHAR(64),
    message_type INTEGER NOT NULL,
    content TEXT,
    status INTEGER DEFAULT 1,
    send_time BIGINT,
    server_time BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 复合主键：会话ID + 序列号
    PRIMARY KEY (conversation_id, seq)
);

-- 4. 创建新的 message_sequences 表（基于会话ID）
CREATE TABLE message_sequences (
    id VARCHAR(64) PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL,
    max_seq BIGINT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_conversation_id (conversation_id)
);

-- 5. 创建索引
-- 会话内幂等索引：防止客户端在同一会话内重复发送
CREATE UNIQUE INDEX idx_messages_conv_client_msg ON messages(conversation_id, client_msg_id);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_send_time ON messages(send_time);

-- 6. 从备份恢复数据（如果需要）
-- 注意：只在数据已经有正确的 seq 的情况下执行
-- INSERT INTO messages 
-- SELECT conversation_id, seq, client_msg_id, sender_id, receiver_id, 
--        group_id, message_type, content, status, send_time, server_time,
--        created_at, updated_at
-- FROM messages_backup
-- WHERE seq IS NOT NULL AND seq > 0;

-- 7. 验证
SELECT 'Migration completed. Please verify data before dropping backup tables.' AS status;
SELECT COUNT(*) as messages_backup_count FROM messages_backup;
SELECT COUNT(*) as messages_new_count FROM messages;
SELECT COUNT(*) as sequences_backup_count FROM message_sequences_backup;
SELECT COUNT(*) as sequences_new_count FROM message_sequences;

-- 8. 清理备份（确认无误后执行）
-- DROP TABLE messages_backup;
-- DROP TABLE message_sequences_backup;

