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

	// 清空数据的表列表
	tables := []string{
		"messages",
		"message_sequences",
		"message_read_receipts",
		"conversations",
		"user_sessions",
		"online_status",
		"friends",
		"friend_requests",
	}

	fmt.Println("\n🗑️  开始清空数据...")

	// 清空每个表
	for _, table := range tables {
		sql := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("⚠️  清空表 %s 失败: %v (表可能不存在)", table, err)
		} else {
			fmt.Printf("   ✅ 已清空表: %s\n", table)
		}
	}

	fmt.Println("\n✅ 数据清空完成！")
	fmt.Println("⚠️  注意：用户表 (users) 未清空，如需清空请手动执行")
}

