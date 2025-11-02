package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Platform string `json:"platform"`
	IssuedAt int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID, platform, secret string, expireHours int) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Platform:  platform,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Duration(expireHours) * time.Hour).Unix(),
	}

	// Header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Payload
	payloadJSON, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Signature
	message := headerB64 + "." + payloadB64
	signature := hmacSHA256(message, secret)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return message + "." + signatureB64, nil
}

// ValidateToken 验证 JWT Token
func ValidateToken(token, secret string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	// 验证签名
	message := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	expectedSignature := hmacSHA256(message, secret)
	if !hmac.Equal(signature, expectedSignature) {
		return nil, errors.New("invalid signature")
	}

	// 解析 payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, errors.New("invalid claims")
	}

	// 检查过期时间
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func hmacSHA256(message, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return h.Sum(nil)
}

