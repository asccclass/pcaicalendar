package main

import "fmt"

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
			fmt.Println(msg)
		}
		fmt.Println("💡 若要強制執行，請增加參數 --force=true 或告知助理強制排入。")
	}

	return conflictFound
}
