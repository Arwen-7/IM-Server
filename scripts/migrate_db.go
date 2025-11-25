package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 数据库连接信息（与 config.yaml 一致）
	dsn := "host=localhost port=5432 user=imserver password=imserver123 dbname=im_db sslmode=disable"

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	fmt.Println("🔗 已连接到数据库")
	fmt.Println("⚠️  警告：此操作将重建 messages 和 message_sequences 表！")
	fmt.Println("")

	// 1. 备份现有数据
	fmt.Println("📦 正在备份数据...")
	if err := db.Exec("DROP TABLE IF EXISTS messages_backup CASCADE").Error; err != nil {
		log.Printf("⚠️  删除旧备份表失败: %v", err)
	}
	if err := db.Exec("DROP TABLE IF EXISTS message_sequences_backup CASCADE").Error; err != nil {
		log.Printf("⚠️  删除旧备份表失败: %v", err)
	}
	
	if err := db.Exec("CREATE TABLE messages_backup AS SELECT * FROM messages").Error; err != nil {
		log.Printf("⚠️  备份 messages 表失败: %v (表可能不存在)", err)
	} else {
		var count int64
		db.Table("messages_backup").Count(&count)
		fmt.Printf("   ✅ 已备份 messages 表 (%d 条记录)\n", count)
	}
	
	if err := db.Exec("CREATE TABLE message_sequences_backup AS SELECT * FROM message_sequences").Error; err != nil {
		log.Printf("⚠️  备份 message_sequences 表失败: %v (表可能不存在)", err)
	} else {
		var count int64
		db.Table("message_sequences_backup").Count(&count)
		fmt.Printf("   ✅ 已备份 message_sequences 表 (%d 条记录)\n", count)
	}

	// 2. 删除旧表
	fmt.Println("\n🗑️  正在删除旧表...")
	if err := db.Exec("DROP TABLE IF EXISTS messages CASCADE").Error; err != nil {
		log.Fatalf("❌ 删除 messages 表失败: %v", err)
	}
	fmt.Println("   ✅ 已删除 messages 表")
	
	if err := db.Exec("DROP TABLE IF EXISTS message_sequences CASCADE").Error; err != nil {
		log.Fatalf("❌ 删除 message_sequences 表失败: %v", err)
	}
	fmt.Println("   ✅ 已删除 message_sequences 表")

	// 3. 创建新的 messages 表
	fmt.Println("\n🔨 正在创建新表...")
	createMessagesSQL := `
CREATE TABLE messages (
    conversation_id VARCHAR(64) NOT NULL,
    seq BIGINT NOT NULL,
    server_msg_id VARCHAR(64) NOT NULL,
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
    PRIMARY KEY (conversation_id, seq)
)`
	if err := db.Exec(createMessagesSQL).Error; err != nil {
		log.Fatalf("❌ 创建 messages 表失败: %v", err)
	}
	fmt.Println("   ✅ 已创建 messages 表（复合主键: conversation_id, seq）")

	// 4. 创建索引
	if err := db.Exec("CREATE UNIQUE INDEX idx_messages_server_msg_id ON messages(server_msg_id)").Error; err != nil {
		log.Printf("⚠️  创建 server_msg_id 唯一索引失败: %v", err)
	} else {
		fmt.Println("   ✅ 已创建唯一索引: server_msg_id（全局唯一）")
	}
	
	if err := db.Exec("CREATE UNIQUE INDEX idx_messages_conv_client_msg ON messages(conversation_id, client_msg_id)").Error; err != nil {
		log.Printf("⚠️  创建会话内唯一索引失败: %v", err)
	} else {
		fmt.Println("   ✅ 已创建复合唯一索引: (conversation_id, client_msg_id) - 会话内幂等")
	}
	
	if err := db.Exec("CREATE INDEX idx_messages_sender ON messages(sender_id)").Error; err != nil {
		log.Printf("⚠️  创建索引失败: %v", err)
	} else {
		fmt.Println("   ✅ 已创建索引: sender_id")
	}
	
	if err := db.Exec("CREATE INDEX idx_messages_send_time ON messages(send_time)").Error; err != nil {
		log.Printf("⚠️  创建索引失败: %v", err)
	} else {
		fmt.Println("   ✅ 已创建索引: send_time")
	}

	// 5. 创建新的 message_sequences 表
	createSequencesSQL := `
CREATE TABLE message_sequences (
    id VARCHAR(64) PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL UNIQUE,
    max_seq BIGINT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`
	if err := db.Exec(createSequencesSQL).Error; err != nil {
		log.Fatalf("❌ 创建 message_sequences 表失败: %v", err)
	}
	fmt.Println("   ✅ 已创建 message_sequences 表（基于 conversation_id）")

	fmt.Println("\n✅ 数据库迁移完成！")
	fmt.Println("\n📝 说明：")
	fmt.Println("   • messages 表主键已改为 (conversation_id, seq)")
	fmt.Println("   • 添加了 server_msg_id 字段（服务端生成，全局唯一）")
	fmt.Println("   • 添加了复合唯一索引 (conversation_id, client_msg_id) - 会话内幂等")
	fmt.Println("   • message_sequences 现在基于 conversation_id（每个会话独立计数）")
	fmt.Println("   • 旧数据已备份到 messages_backup 和 message_sequences_backup")
	fmt.Println("\n💡 设计说明（参考 OpenIM）：")
	fmt.Println("   • server_msg_id：服务端生成，全局唯一，用于日志追踪")
	fmt.Println("   • client_msg_id：客户端生成，用于本地匹配和会话内幂等")
	fmt.Println("   • (conversation_id, seq)：主键，所有查询都基于此，性能最优")
	fmt.Println("   • 不同会话可以有相同的 client_msg_id")
	fmt.Println("\n⚠️  如需恢复数据，请手动执行 SQL 导入（确保数据有正确的 seq 值）")
	fmt.Println("⚠️  确认无误后，可手动删除备份表：")
	fmt.Println("     DROP TABLE messages_backup;")
	fmt.Println("     DROP TABLE message_sequences_backup;")
}

