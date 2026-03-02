package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// --- 核心功能：更新行程 ---
func runUpdateMode(from, to, summary, calInput, eventID, rrule, location string, force bool) {
	targetID := resolveCalendarID(calInput)
	actualEventID := eventID

	// 檢查 eventID 是否真的是一個合法的 ID (不含空白或特殊字元)
	if eventID != "" && (!isAlphanumericOrHyphen(eventID) || strings.Contains(eventID, " ")) {
		fmt.Printf("⚠️ 警告：提供的 event parameter [%s] 看起來不像合法的 ID，自動轉為標題搜尋。\n", eventID)
		if summary == "" {
			summary = eventID // 將它當作 summary 搜尋
		}
		actualEventID = ""
	}

	// 1. 如果沒有 ID，執行智慧搜尋
	if actualEventID == "" {
		if summary == "" {
			log.Fatal("❌ 錯誤：更新時若無 ID，必須提供原標題以供搜尋。")
		}
		fmt.Printf("🔍 正在搜尋要更新的行程: [%s]...\n", summary)
		events, _ := getEventsJSON(targetID, from, to)

		var matches []Event
		for _, e := range events {
			if strings.Contains(strings.ToLower(e.Summary), strings.ToLower(summary)) {
				matches = append(matches, e)
			}
		}

		if len(matches) == 0 {
			log.Fatalf("❌ 找不到匹配 [%s] 的行程。", summary)
		} else if len(matches) > 1 {
			log.Fatalf("⚠️ 找到多個匹配行程，請提供事件 ID 以確保更新正確。")
		}
		actualEventID = matches[0].ID
	}

	// 1.5 新增：更新前檢查衝突
	// 如果有提供新的完整時間區間且未強制執行，則檢查衝突
	if !force && from != "" && to != "" {
		if hasConflict(targetID, from, to, actualEventID) {
			os.Exit(1) // 發現衝突，回傳非零退出碼
		}
	}

	// 2. 執行 gog update
	executable := getGogPath()
	args := []string{"calendar", "update", targetID, actualEventID}

	// 僅加入有變動的參數 (假設 from/to 是要更新的新時間)
	if summary != "" {
		args = append(args, "--summary", summary)
	}
	if from != "" {
		args = append(args, "--from", from)
	}
	if to != "" {
		args = append(args, "--to", to)
	}
	if rrule != "" {
		args = append(args, "--rrule", rrule)
	}
	if location != "" {
		args = append(args, "--location", location)
	}

	fmt.Printf("🚀 正在更新事件 [%s]...\n", actualEventID)

	cmd := exec.Command(executable, args...)
	cmd.Dir = filepath.Dir(executable)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("❌ 更新失敗: %v\n回傳: %s", err, string(output))
	}
	fmt.Println("✅ 行程更新成功！")
}
