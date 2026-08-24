package trial

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Fingerprint 计算试验指纹（SHA-256），用于幂等去重：
// 相同 code + cable_name + voltage_class 视为重复试验。
func Fingerprint(code, cableName, voltageClass string) string {
	h := sha256.New()
	h.Write([]byte(strings.TrimSpace(code)))
	h.Write([]byte{0})
	h.Write([]byte(strings.TrimSpace(cableName)))
	h.Write([]byte{0})
	h.Write([]byte(strings.TrimSpace(voltageClass)))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidCode 校验试验编号非空且长度受限（1~64）。
func ValidCode(code string) bool {
	c := strings.TrimSpace(code)
	return len(c) >= 1 && len(c) <= 64
}
