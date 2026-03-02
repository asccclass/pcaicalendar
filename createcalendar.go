package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// --- 新增活動的函數 ---
func createEvent(calID, summary, from, to, rrule, reminders, location string) error {
	if summary == "" {
		log.Fatal("❌ 錯誤：必須提供行程標題 (--summary)")
	}
	if from == "" || to == "" {
		log.Fatal("❌ 錯誤：必須提供開始與結束時間 (--from, --to)")
	}
	calID = strings.TrimSpace(calID)
	// 透過智慧識別取得正確 ID
	targetID := resolveCalendarID(calID)
	executable := getGogPath()

	args := []string{"calendar", "create", targetID}
	args = append(args, "--summary", summary)
	args = append(args, "--from", from)
	args = append(args, "--to", to)
	rrule = strings.TrimSpace(rrule)
	if rrule != "" {
		args = append(args, "--rrule", rrule)
	}
	location = strings.TrimSpace(location)
	if location != "" {
		args = append(args, "--location", location)
	}

	// 處理多個提醒設定
	if reminders != "" {
		// 假設輸入格式為 "email:3d,popup:30m"
		// (註：實際程式碼中 strings 需移至上方 import 區)
		parts := strings.Split(reminders, ",")
		for _, p := range parts {
			args = append(args, "--reminder", strings.TrimSpace(p))
		}
	}
	fmt.Printf("🛠️  準備執行指令: %s %s\n", executable, strings.Join(args, " "))
	cmd := exec.Command(executable, args...)
	cmd.Env = os.Environ() // 繼承父程序的環境變數
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	outStr := string(output)
	if strings.Contains(strings.ToLower(outStr), "error") || strings.Contains(outStr, "400") {
		fmt.Printf("⚠️ 指令看似成功執行，但 Google 回傳了錯誤：\n%s\n", outStr)
	}
	return nil
}
