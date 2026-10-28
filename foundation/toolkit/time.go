package toolkit

import (
	"fmt"
	"time"
)

// 常用时间格式常量，取值为 time 包的参考时间布局，
// 可作为 [Format]、[FormatNow] 与 [Parse] 的 layout 参数。
const (
	// TimeFormatDate 日期布局，形如 2006-01-02。
	TimeFormatDate = "2006-01-02"
	// TimeFormatDateTime 日期时间布局，形如 2006-01-02 15:04:05。
	TimeFormatDateTime = "2006-01-02 15:04:05"
	// TimeFormatTime 时间布局，形如 15:04:05。
	TimeFormatTime = "15:04:05"
	// TimeFormatISO8601 ISO 8601 带时区布局，
	// 形如 2006-01-02T15:04:05Z07:00。
	TimeFormatISO8601 = "2006-01-02T15:04:05Z07:00"
	// TimeFormatUnix 该值不是 time 包参考时间的合法布局，传入
	// [Format] 或 [Parse] 无法得到 Unix 时间戳，保留仅为兼容既有
	// 引用；需要 Unix 时间戳请使用 [NowUnix] 等函数。
	TimeFormatUnix = "1136239445"
)

// Now 获取当前本地时间。
//
// 等价于 time.Now()，语义上便于与其他工具函数组合使用。
//
// 返回值:
//   - 当前本地时间。
func Now() time.Time {
	return time.Now()
}

// NowUnix 获取当前时间的 Unix 时间戳（秒）。
//
// 返回值:
//   - 当前 Unix 时间戳（秒）。
func NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixMilli 获取当前时间的 Unix 时间戳（毫秒）。
//
// 返回值:
//   - 当前 Unix 时间戳（毫秒）。
func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}

// Format 格式化时间。
//
// 语义与 time.Time.Format 一致，layout 可选用本包的
// [TimeFormatDate] 等布局常量。
//
// 参数说明:
//   - t: 要格式化的时间。
//   - layout: 格式模板。
//
// 返回值:
//   - 格式化后的时间字符串。
func Format(t time.Time, layout string) string {
	return t.Format(layout)
}

// FormatNow 格式化当前本地时间。
//
// 相当于 [Now] 后直接按 layout 格式化，适合日志时间戳等场景。
//
// 参数说明:
//   - layout: 格式模板。
//
// 返回值:
//   - 格式化后的当前时间字符串。
func FormatNow(layout string) string {
	return time.Now().Format(layout)
}

// Parse 解析时间字符串。
//
// 语义与 time.Parse 一致，解析结果为 UTC 时区（无时区信息时）；
// 解析失败时返回零值时间与错误。
//
// 参数说明:
//   - layout: 格式模板。
//   - value: 时间字符串。
//
// 返回值:
//   - 解析后的时间。
//   - 解析失败时返回对应错误。
func Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

// MustParse 解析时间字符串，如果解析失败则 panic。
//
// 适合初始化阶段解析已保证合法的常量时间；来源不可靠的输入
// 请使用 [Parse] 并处理错误。
//
// 参数说明:
//   - layout: 格式模板。
//   - value: 时间字符串。
//
// 返回值:
//   - 解析后的时间；解析失败时 panic 并携带错误信息。
func MustParse(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(fmt.Sprintf("解析时间失败: %v", err))
	}
	return t
}

// FromUnix 将 Unix 时间戳（秒）转换为本地时间。
//
// 语义与 time.Unix(unix, 0) 一致，结果位于本地时区。
//
// 参数说明:
//   - unix: Unix 时间戳（秒）。
//
// 返回值:
//   - 对应的本地时间。
func FromUnix(unix int64) time.Time {
	return time.Unix(unix, 0)
}

// FromUnixMilli 将 Unix 时间戳（毫秒）转换为本地时间。
//
// 语义与 time.UnixMilli 一致，结果位于本地时区。
//
// 参数说明:
//   - unixMilli: Unix 时间戳（毫秒）。
//
// 返回值:
//   - 对应的本地时间。
func FromUnixMilli(unixMilli int64) time.Time {
	return time.UnixMilli(unixMilli)
}

// IsToday 检查时间是否为今天。
//
// 以本机当前本地时间为基准，按年份与年内天数（YearDay）比较，
// 不涉及时区换算。
//
// 参数说明:
//   - t: 要检查的时间。
//
// 返回值:
//   - 如果是今天返回 true，否则返回 false。
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}

// IsYesterday 检查时间是否为昨天。
//
// 以本机当前本地时间减一天为基准，按年份与年内天数（YearDay）
// 比较。
//
// 参数说明:
//   - t: 要检查的时间。
//
// 返回值:
//   - 如果是昨天返回 true，否则返回 false。
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay()
}

// IsWeekend 检查时间是否为周末。
//
// 仅按 t 自身的星期判断（周六或周日），不做时区换算。
//
// 参数说明:
//   - t: 要检查的时间。
//
// 返回值:
//   - 如果是周六或周日返回 true，否则返回 false。
func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// StartOfDay 获取指定日期的开始时间（00:00:00）。
//
// 结果沿用 t 的时区信息。
//
// 参数说明:
//   - t: 指定日期。
//
// 返回值:
//   - 当天的开始时间。
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取指定日期的结束时间（23:59:59.999999999）。
//
// 结果沿用 t 的时区信息；纳秒级末值与次日零点之间无间隙。
//
// 参数说明:
//   - t: 指定日期。
//
// 返回值:
//   - 当天的结束时间。
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek 获取指定日期所在周的开始时间（周一 00:00:00）。
//
// 以周一为一周的开始，周日归入上一周；结果沿用 t 的时区信息。
//
// 参数说明:
//   - t: 指定日期。
//
// 返回值:
//   - 本周周一的开始时间。
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 将周日调整为 7
	}
	return StartOfDay(t.AddDate(0, 0, 1-weekday))
}

// EndOfWeek 获取指定日期所在周的结束时间（周日 23:59:59.999999999）。
//
// 即 [StartOfWeek] 结果加六天的当日末值，结果沿用 t 的时区信息。
//
// 参数说明:
//   - t: 指定日期。
//
// 返回值:
//   - 本周周日的结束时间。
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth 获取指定日期所在月的开始时间（当月 1 日 00:00:00）。
//
// 结果沿用 t 的时区信息。
//
// 参数说明:
//   - t: 指定日期。
//
// 返回值:
//   - 本月的开始时间。
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取指定日期所在月的结束时间（当月最后一天 23:59:59.999999999）。
//
// 即 [StartOfMonth] 结果加一个月后退回一天的当日末值，自然处理
// 大小月与闰年；结果沿用 t 的时区信息。
//
// 参数说明:
//   - t: 指定日期。
//
// 返回值:
//   - 本月的结束时间。
func EndOfMonth(t time.Time) time.Time {
	return EndOfDay(StartOfMonth(t).AddDate(0, 1, -1))
}

// DiffDays 计算两个时间之间的天数差。
//
// 结果为 t1.Sub(t2) 的小时数除以 24 后向零截断的整数，
// 是固定 24 小时制的时长差而非日历日差（跨越夏令时切换时
// 与自然日数可能不一致）；t1 早于 t2 时结果为负。
//
// 参数说明:
//   - t1: 第一个时间。
//   - t2: 第二个时间。
//
// 返回值:
//   - 天数差（t1 - t2）。
func DiffDays(t1, t2 time.Time) int {
	return int(t1.Sub(t2).Hours() / 24)
}

// DiffHours 计算两个时间之间的小时差。
//
// 结果为 float64，可表示不足一小时的部分；t1 早于 t2 时为负。
//
// 参数说明:
//   - t1: 第一个时间。
//   - t2: 第二个时间。
//
// 返回值:
//   - 小时差（t1 - t2）。
func DiffHours(t1, t2 time.Time) float64 {
	return t1.Sub(t2).Hours()
}

// DiffMinutes 计算两个时间之间的分钟差。
//
// 结果为 float64，可表示不足一分钟的部分；t1 早于 t2 时为负。
//
// 参数说明:
//   - t1: 第一个时间。
//   - t2: 第二个时间。
//
// 返回值:
//   - 分钟差（t1 - t2）。
func DiffMinutes(t1, t2 time.Time) float64 {
	return t1.Sub(t2).Minutes()
}

// Age 根据生日计算年龄。
//
// 以本机当前本地时间按年份差计算，并通过年内天数（YearDay）
// 粗略判断今年生日是否已过，未过则减一；不比较具体时刻，
// 闰年 2 月 29 日生日会存在一日偏差。生日晚于当前时间时
// 结果可能为负。
//
// 参数说明:
//   - birthday: 生日。
//
// 返回值:
//   - 年龄（周岁近似值）。
func Age(birthday time.Time) int {
	now := time.Now()
	age := now.Year() - birthday.Year()

	// 如果今年的生日还没过，年龄减 1
	if now.YearDay() < birthday.YearDay() {
		age--
	}

	return age
}
