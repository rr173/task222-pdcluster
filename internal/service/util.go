package service

import "time"

// nowISO 返回 UTC 时间的 RFC3339Nano 字符串（与 store 层时间格式一致）。
func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }
