package main

import (
	"fmt"
	"time"
)

// --- 核心功能：尋找並建議空檔 ---
func suggestSlots(calID, fromStr, toStr string) {
	const layout = "2006-01-02T15:04:05-07:00"
	reqStart, _ := time.Parse(layout, fromStr)
	reqEnd, _ := time.Parse(layout, toStr)
	duration := reqEnd.Sub(reqStart)

	fmt.Printf("🔍 正在為您尋找足以容納 %v 的其他空檔...\n", duration)

	// 設定掃描範圍：當天 08:00 ~ 22:00
	datePart := reqStart.Format("2006-01-02")
	scanStart, _ := time.Parse(layout, datePart+"T08:00:00+08:00")
	scanEnd, _ := time.Parse(layout, datePart+"T22:00:00+08:00")

	// 抓取當天所有行程
	events, _ := getEventsJSON(calID, scanStart.Format(layout), scanEnd.Format(layout))

	// 將行程轉換為時間區間
	type interval struct{ start, end time.Time }
	var busy []interval
	for _, e := range events {
		s, _ := time.Parse(layout, e.Start.DateTime)
		eTime, _ := time.Parse(layout, e.End.DateTime)
		if !s.IsZero() && !eTime.IsZero() {
			busy = append(busy, interval{s, eTime})
		}
	}

	// 尋找空檔
	curr := scanStart
	foundCount := 0
	fmt.Println("💡 建議的可更換時段：")

	for curr.Add(duration).Before(scanEnd) && foundCount < 3 {
		isFree := true
		potentialEnd := curr.Add(duration)

		for _, b := range busy {
			// 檢查重疊 (Overlap logic)
			if curr.Before(b.end) && potentialEnd.After(b.start) {
				isFree = false
				curr = b.end // 直接跳到該忙碌時段結束後繼續找
				break
			}
		}

		if isFree {
			// 避開原本衝突的時段
			if !(curr.Before(reqEnd) && potentialEnd.After(reqStart)) {
				fmt.Printf("   ✅ %s ~ %s\n", curr.Format("15:04"), potentialEnd.Format("15:04"))
				foundCount++
			}
			curr = curr.Add(30 * time.Minute) // 每次移動 30 分鐘尋找下一個
		}
	}

	if foundCount == 0 {
		fmt.Println("   (今天 08:00~22:00 之間似乎沒有足夠的連續空檔了)")
	}
}

// --- 核心功能：衝突檢查 ---
func hasConflict(calID, from, to, excludeEventID string) bool {
	fmt.Printf("🛡️  正在檢查時段衝突 (%s ~ %s)...\n", from, to)

	// 抓取該時段現有的行程
	events, err := getEventsJSON(calID, from, to)
	if err != nil || len(events) == 0 {
		return false // 沒有行程，無衝突
	}

	conflictFound := false
	var conflictMsgs []string

	for _, e := range events {
		if excludeEventID != "" && e.ID == excludeEventID {
			continue // Skip the event being updated
		}
		timeDesc := e.Start.DateTime
		if timeDesc == "" {
			timeDesc = e.Start.Date + " [全天]"
		}
		conflictMsgs = append(conflictMsgs, fmt.Sprintf("   🚫 [%s] - %s", e.Summary, timeDesc))
		conflictFound = true
	}

	if conflictFound {
		fmt.Println("⚠️  偵測到行程衝突！以下行程已佔用此時段：")
		for _, msg := range conflictMsgs {
			fmt.Printf("   🚫 %s\n", msg)
		}
		// 🌟 執行空檔建議邏輯
		suggestSlots(calID, from, to)
		fmt.Println("💡 若要強制執行，請增加參數 --force=true 或告知助理強制排入。")
	}

	return conflictFound
}
