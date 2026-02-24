package pack

import (
	"fmt"
	"time"
)

// 常用时间格式常量
const (
	TimeFormatDate     = "2006-01-02"
	TimeFormatDateTime = "2006-01-02 15:04:05"
	TimeFormatTime     = "15:04:05"
	TimeFormatISO8601  = "2006-01-02T15:04:05Z07:00"
	TimeFormatUnix     = "1136239445"
)

// Timer 时间工具结构体，提供常用的时间处理方法。
//
// 使用方式：
//
//	xUtil.Timer().Now()
//	xUtil.Timer().Format(t, layout)
//	xUtil.Timer().FromUnix(ts)
type Timer struct{}

// Now 获取当前时间。
//
// 返回值:
//   - 当前时间
func (Timer) Now() time.Time {
	return time.Now()
}

// NowUnix 获取当前时间的 Unix 时间戳（秒）。
//
// 返回值:
//   - 当前 Unix 时间戳
func (Timer) NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixMilli 获取当前时间的 Unix 时间戳（毫秒）。
//
// 返回值:
//   - 当前 Unix 时间戳（毫秒）
func (Timer) NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}

// Format 格式化时间。
//
// 参数说明:
//   - t: 要格式化的时间
//   - layout: 格式模板
//
// 返回值:
//   - 格式化后的时间字符串
func (Timer) Format(t time.Time, layout string) string {
	return t.Format(layout)
}

// FormatNow 格式化当前时间。
//
// 参数说明:
//   - layout: 格式模板
//
// 返回值:
//   - 格式化后的当前时间字符串
func (Timer) FormatNow(layout string) string {
	return time.Now().Format(layout)
}

// Parse 解析时间字符串。
//
// 参数说明:
//   - layout: 格式模板
//   - value: 时间字符串
//
// 返回值:
//   - 解析后的时间和错误信息
func (Timer) Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

// MustParse 解析时间字符串，如果解析失败则 panic。
//
// 参数说明:
//   - layout: 格式模板
//   - value: 时间字符串
//
// 返回值:
//   - 解析后的时间
func (Timer) MustParse(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(fmt.Sprintf("解析时间失败: %v", err))
	}
	return t
}

// FromUnix 将 Unix 时间戳转换为时间。
//
// 参数说明:
//   - unix: Unix 时间戳（秒）
//
// 返回值:
//   - 对应的时间
func (Timer) FromUnix(unix int64) time.Time {
	return time.Unix(unix, 0)
}

// FromUnixMilli 将 Unix 时间戳（毫秒）转换为时间。
//
// 参数说明:
//   - unixMilli: Unix 时间戳（毫秒）
//
// 返回值:
//   - 对应的时间
func (Timer) FromUnixMilli(unixMilli int64) time.Time {
	return time.UnixMilli(unixMilli)
}

// IsToday 检查时间是否为今天。
//
// 参数说明:
//   - t: 要检查的时间
//
// 返回值:
//   - 如果是今天返回 true，否则返回 false
func (Timer) IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}

// IsYesterday 检查时间是否为昨天。
//
// 参数说明:
//   - t: 要检查的时间
//
// 返回值:
//   - 如果是昨天返回 true，否则返回 false
func (Timer) IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay()
}

// IsWeekend 检查时间是否为周末。
//
// 参数说明:
//   - t: 要检查的时间
//
// 返回值:
//   - 如果是周末返回 true，否则返回 false
func (Timer) IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// StartOfDay 获取指定日期的开始时间（00:00:00）。
//
// 参数说明:
//   - t: 指定日期
//
// 返回值:
//   - 当天的开始时间
func (Timer) StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取指定日期的结束时间（23:59:59.999999999）。
//
// 参数说明:
//   - t: 指定日期
//
// 返回值:
//   - 当天的结束时间
func (Timer) EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek 获取指定日期所在周的开始时间（周一 00:00:00）。
//
// 参数说明:
//   - t: 指定日期
//
// 返回值:
//   - 本周的开始时间
func (ti Timer) StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 将周日调整为 7
	}
	return ti.StartOfDay(t.AddDate(0, 0, 1-weekday))
}

// EndOfWeek 获取指定日期所在周的结束时间（周日 23:59:59.999999999）。
//
// 参数说明:
//   - t: 指定日期
//
// 返回值:
//   - 本周的结束时间
func (ti Timer) EndOfWeek(t time.Time) time.Time {
	return ti.EndOfDay(ti.StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth 获取指定日期所在月的开始时间。
//
// 参数说明:
//   - t: 指定日期
//
// 返回值:
//   - 本月的开始时间
func (Timer) StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取指定日期所在月的结束时间。
//
// 参数说明:
//   - t: 指定日期
//
// 返回值:
//   - 本月的结束时间
func (ti Timer) EndOfMonth(t time.Time) time.Time {
	return ti.EndOfDay(ti.StartOfMonth(t).AddDate(0, 1, -1))
}

// DiffDays 计算两个日期之间的天数差。
//
// 参数说明:
//   - t1: 第一个时间
//   - t2: 第二个时间
//
// 返回值:
//   - 天数差（t1 - t2）
func (Timer) DiffDays(t1, t2 time.Time) int {
	return int(t1.Sub(t2).Hours() / 24)
}

// DiffHours 计算两个时间之间的小时差。
//
// 参数说明:
//   - t1: 第一个时间
//   - t2: 第二个时间
//
// 返回值:
//   - 小时差（t1 - t2）
func (Timer) DiffHours(t1, t2 time.Time) float64 {
	return t1.Sub(t2).Hours()
}

// DiffMinutes 计算两个时间之间的分钟差。
//
// 参数说明:
//   - t1: 第一个时间
//   - t2: 第二个时间
//
// 返回值:
//   - 分钟差（t1 - t2）
func (Timer) DiffMinutes(t1, t2 time.Time) float64 {
	return t1.Sub(t2).Minutes()
}

// Age 计算年龄（根据生日）。
//
// 参数说明:
//   - birthday: 生日
//
// 返回值:
//   - 年龄
func (Timer) Age(birthday time.Time) int {
	now := time.Now()
	age := now.Year() - birthday.Year()

	// 如果今年的生日还没过，年龄减 1
	if now.YearDay() < birthday.YearDay() {
		age--
	}

	return age
}
