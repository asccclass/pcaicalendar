package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func executeDelete(calID, eventID string) {
	executable := getGogPath()
	args := []string{"calendar", "delete", calID, eventID, "--force"}

	cmd := exec.Command(executable, args...)
	cmd.Dir = filepath.Dir(executable)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("❌ 刪除失敗: %v\n回傳內容: %s", err, string(output))
	}
	fmt.Printf("✅ 已成功刪除事件 ID: %s\n", eventID)
}

// --- 核心功能：刪除行程 ---
func runDeleteMode(from, to, summary, calInput, eventID string) {
	// 檢查 eventID 是否真的是一個合法的 ID (不含空白或特殊字元)
	if eventID != "" && (!isAlphanumericOrHyphen(eventID) || strings.Contains(eventID, " ")) {
		fmt.Printf("⚠️ 警告：提供的 event parameter [%s] 看起來不像合法的 ID，自動轉為標題搜尋。\n", eventID)
		if summary == "" {
			summary = eventID // 將它當作 summary 搜尋
		}
		eventID = ""
	}

	// A. 如果已經有 ID，直接刪除
	if eventID != "" {
		executeDelete(resolveCalendarID(calInput), eventID)
		return
	}
	// B. 如果沒有 ID，啟動「搜尋並比對」流程
	if summary == "" {
		log.Fatal("❌ 錯誤：未提供事件 ID 時，必須提供 --summary 以供比對刪除。")
	}
	// 1. 取得所有 owner 日曆
	cals, _ := getCalendarsJSON()
	var candidates []struct {
		CalID   string
		EventID string
		Title   string
		Time    string
	}
	// 2. 遍歷所有日曆搜尋事件
	for _, c := range cals {
		if c.AccessRole != "owner" {
			continue
		}

		// 如果使用者有指定某個日曆，則只搜尋該日曆
		if calInput != "" && !strings.Contains(strings.ToLower(c.Summary), strings.ToLower(calInput)) && c.ID != calInput {
			continue
		}

		events, _ := getEventsJSON(c.ID, from, to)
		for _, e := range events {
			// 標題模糊匹配
			if strings.Contains(strings.ToLower(e.Summary), strings.ToLower(summary)) {
				tStr := e.Start.DateTime
				if tStr == "" {
					tStr = e.Start.Date + " [全天]"
				}
				candidates = append(candidates, struct {
					CalID, EventID, Title, Time string
				}{c.ID, e.ID, e.Summary, tStr})
			}
		}
	}

	// 3. 判斷搜尋結果
	if len(candidates) == 0 {
		fmt.Printf("❌ 找不到匹配 [%s] 的行程。\n", summary)
		return
	}

	if len(candidates) > 1 {
		fmt.Println("⚠️  找到多個匹配行程，請提供更精確的時間或事件 ID：")
		for _, cand := range candidates {
			fmt.Printf("   - ID: %s | 時間: %s | 標題: %s\n", cand.EventID, cand.Time, cand.Title)
		}
		return
	}

	// 4. 找到唯一匹配，執行刪除
	target := candidates[0]
	fmt.Printf("🎯 找到唯一匹配：[%s] (%s)\n", target.Title, target.Time)
	executeDelete(target.CalID, target.EventID)
}
