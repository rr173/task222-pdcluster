package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewID 生成一个 16 字节随机十六进制 ID（不含连字符）。
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rand: %v", err))
	}
	return hex.EncodeToString(b)
}

// isUniqueViolation 判断 SQLite 返回的约束冲突错误。
// modernc.org/sqlite 驱动的唯一约束错误信息包含 "UNIQUE constraint failed"。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed") || contains(msg, "constraint failed")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
