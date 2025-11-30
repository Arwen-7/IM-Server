-- 清空消息相关数据（保留用户和群组）
TRUNCATE TABLE messages CASCADE;
TRUNCATE TABLE message_sequences CASCADE;
TRUNCATE TABLE conversations CASCADE;

SELECT 'Messages cleared successfully!' AS result;
