package config

import "time"

// CalculateRemaining 根据额度类型计算剩余额度
func CalculateRemaining(key APIKey) float64 {
	remaining := key.QuotaLimit - key.QuotaUsed
	if remaining < 0 {
		remaining = 0
	}
	if ShouldReset(key.QuotaType, key.LastChecked) {
		remaining = key.QuotaLimit
	}
	return remaining
}

// ShouldReset 判断额度是否需要按照周期重置
func ShouldReset(quotaType string, lastChecked time.Time) bool {
	now := time.Now()
	switch quotaType {
	case QuotaDaily:
		return !isSameDay(now, lastChecked)
	case QuotaWeekly:
		return !isSameWeek(now, lastChecked)
	case QuotaMonthly:
		return !isSameMonth(now, lastChecked)
	case QuotaYearly:
		return !isSameYear(now, lastChecked)
	default:
		return false
	}
}

func isSameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func isSameWeek(a, b time.Time) bool {
	ay, aw := a.ISOWeek()
	by, bw := b.ISOWeek()
	return ay == by && aw == bw
}

func isSameMonth(a, b time.Time) bool {
	ay, am, _ := a.Date()
	by, bm, _ := b.Date()
	return ay == by && am == bm
}

// isSameYear 判断两个时间是否处于同一年
func isSameYear(a, b time.Time) bool {
	return a.Year() == b.Year()
}
