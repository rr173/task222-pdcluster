package model

import "errors"

// 领域错误哨兵，service 层据此映射到 HTTP 状态码。
var (
	// ErrNotFound 目标实体不存在。
	ErrNotFound = errors.New("not found")
	// ErrConflict 唯一键冲突（重复试验/重复脉冲/重复版本等）。
	ErrConflict = errors.New("conflict: duplicate or inconsistent")
	// ErrInvalidState 状态机非法流转（如对封存试验写入、跳过前置状态）。
	ErrInvalidState = errors.New("invalid state transition")
	// ErrInvalidArgument 输入不合法（相位参考缺失、通道未知、时间非单调等）。
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrSealed 对封存（sealed）试验的写入被拒绝。
	ErrSealed = errors.New("trial is sealed")
	// ErrVersionConflict 快照版本冲突（并发发布）。
	ErrVersionConflict = errors.New("snapshot version conflict")
)

// IsNotFound 便捷判断。
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsConflict 便捷判断。
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }

// IsInvalidState 便捷判断。
func IsInvalidState(err error) bool { return errors.Is(err, ErrInvalidState) }
