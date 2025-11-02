package utils

import "time"

// GetCurrentMillis 获取当前时间戳（毫秒）
func GetCurrentMillis() int64 {
	return time.Now().UnixMilli()
}

// GetCurrentTime 获取当前时间
func GetCurrentTime() time.Time {
	return time.Now()
}

// MillisToTime 毫秒时间戳转时间
func MillisToTime(millis int64) time.Time {
	return time.UnixMilli(millis)
}

// TimeToMillis 时间转毫秒时间戳
func TimeToMillis(t time.Time) int64 {
	return t.UnixMilli()
}

