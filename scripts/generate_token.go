package main

import (
	"flag"
	"fmt"
	"os"
)

// 简单的Token生成工具
// 用于测试和开发

func main() {
	userID := flag.String("user", "", "User ID")
	platform := flag.String("platform", "iOS", "Platform")
	secret := flag.String("secret", "your-secret-key-change-in-production", "JWT Secret")
	hours := flag.Int("hours", 720, "Token expire hours")
	
	flag.Parse()
	
	if *userID == "" {
		fmt.Println("Error: user ID is required")
		fmt.Println("Usage: go run generate_token.go -user=user123 -platform=iOS")
		os.Exit(1)
	}
	
	// 简化的Token生成逻辑（实际应该导入crypto包）
	fmt.Printf(`
Token Generator
===============
User ID:    %s
Platform:   %s
Expire:     %d hours

To generate a real token, use:
  import "github.com/arwen/im-server/pkg/crypto"
  token, _ := crypto.GenerateToken("%s", "%s", "%s", %d)

Or use the web interface (coming soon)
`, *userID, *platform, *hours, *userID, *platform, *secret, *hours)
}

