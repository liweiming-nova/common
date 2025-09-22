package lock

import "fmt"

// LockErrorCode 锁错误的“枚举”类型
type LockErrorCode int

const (
	LockErrorNone          LockErrorCode = iota // 0 - 不是锁错误
	LockErrorTimeout                            // 1 - 锁超时
	LockErrorAlreadyLocked                      // 2 - 已被其他进程锁定
	LockErrorNetwork                            // 3 - 网络或 Redis 错误
	LockErrorInvalidKey                         // 4 - 锁 key 无效
	LockErrorReleaseFailed                      // 5 - 释放锁失败
)

// 实现 String() 方法，便于打印
func (c LockErrorCode) String() string {
	switch c {
	case LockErrorTimeout:
		return "LOCK_TIMEOUT"
	case LockErrorAlreadyLocked:
		return "ALREADY_LOCKED"
	case LockErrorNetwork:
		return "NETWORK_ERROR"
	case LockErrorInvalidKey:
		return "INVALID_KEY"
	case LockErrorReleaseFailed:
		return "RELEASE_FAILED"
	default:
		return "NONE"
	}
}

// LockError 自定义错误结构体，携带 code
type LockError struct {
	Code    LockErrorCode
	Message string
}

// 实现 error 接口
func (e *LockError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

// 预定义错误常量（对外暴露）
var (
	ErrLockTimeout    = &LockError{Code: LockErrorTimeout, Message: "lock timeout"}
	ErrAlreadyLocked  = &LockError{Code: LockErrorAlreadyLocked, Message: "already locked by another process"}
	ErrLockNetwork    = &LockError{Code: LockErrorNetwork, Message: "network or redis error"}
	ErrLockInvalidKey = &LockError{Code: LockErrorInvalidKey, Message: "invalid lock key"}
	ErrReleaseFailed  = &LockError{Code: LockErrorReleaseFailed, Message: "release failed"}
)

// IsError 判断是否为锁错误，返回对应的枚举 code（0 表示不是）
func IsError(err error) LockErrorCode {
	if le, ok := err.(*LockError); ok {
		return le.Code
	}
	return LockErrorNone
}
