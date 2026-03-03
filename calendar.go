package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// --- Data Structures for parsing gog output ---

type CalendarListResponse struct {
	Calendars []Calendar `json:"calendars"`
}

type Calendar struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	AccessRole string `json:"accessRole"`
	Primary    bool   `json:"primary"` // 新增：用來識別預設日曆
}

type EventResponse struct {
	Events []Event `json:"events"`
}

type Event struct {
	ID          string `json:"id"` // 🌟 修正點：加入 ID 欄位
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Location    string `json:"location"` // 新增：接收 gog 輸出的地點資訊
	Start       struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"end"`
}

// --- Final Output Structure ---

type FinalOutput struct {
	Creator string           `json:"creator"`
	Events  []FormattedEvent `json:"events"`
}

type FormattedEvent struct {
	ID        string `json:"id"` // 🌟 建議加入，方便 AI 直接獲取 ID
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	EventName string `json:"event_name"`
	Location  string `json:"location"` // 新增：輸出給 LLM 的地點資訊
	Summary   string `json:"summary"`
}

// --- 整合時間處理函數 ---
func processEventTimes(e Event) (string, string) {
	const timeLayout = "2006-01-02 15:04:05"
	var startTime, endTime time.Time

	if e.Start.DateTime != "" {
		startTime, _ = time.Parse(time.RFC3339, e.Start.DateTime)
		endTime, _ = time.Parse(time.RFC3339, e.End.DateTime)
	} else {
		startTime, _ = time.Parse("2006-01-02", e.Start.Date)
		endTime, _ = time.Parse("2006-01-02", e.End.Date)
	}

	// 修正全天事件顯示 (Exclusive End Date 減一天)
	isAllDay := startTime.Hour() == 0 && startTime.Minute() == 0 && startTime.Second() == 0 &&
		endTime.Hour() == 0 && endTime.Minute() == 0 && endTime.Second() == 0

	if isAllDay || (e.Start.DateTime == "" && e.Start.Date != "") {
		correctedEnd := endTime.AddDate(0, 0, -1)
		if correctedEnd.Before(startTime) {
			correctedEnd = startTime
		}
		return startTime.Format(timeLayout), correctedEnd.Format(timeLayout)
	}

	return startTime.Format(timeLayout), endTime.Format(timeLayout)
}

func getGogPath() string {
	// First check environment variable
	if envPath := os.Getenv("GOG_PATH"); envPath != "" {
		if path, err := filepath.Abs(envPath); err == nil {
			return path
		}
	}

	fileName := "gog"
	if runtime.GOOS == "windows" {
		fileName = "gog.exe"
	}

	// Depend on executable location
	if exePath, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exePath), fileName)
	}

	// Fallback
	binDir := filepath.Join("..", "bin")
	path, _ := filepath.Abs(filepath.Join(binDir, fileName))
	return path
}

func getDefaultCalendarID() string {
	cals, err := getCalendarsJSON()
	if err != nil {
		log.Fatalf("無法讀取日曆列表: %v", err)
	}
	for _, c := range cals {
		if c.Primary {
			return c.ID
		}
	}
	// 若找不到標註 primary 的，回傳第一個 owner
	for _, c := range cals {
		if c.AccessRole == "owner" {
			return c.ID
		}
	}
	return "primary" // 最後的 fallback
}

func normalizeDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return ""
	}
	// 1. 分離日期與時間
	parts := strings.Split(dateStr, "T")
	dateStr = parts[0]
	timeStr := ""
	if len(parts) > 1 {
		timeStr = "T" + parts[1]
	}

	// 2. 正規化日期 (處理 2/29 溢出)
	var y, m, d int
	if _, err := fmt.Sscanf(dateStr, "%d-%d-%d", &y, &m, &d); err != nil {
		return dateStr // 格式不符則不處理
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)
	normalizedDate := t.Format("2006-01-02")

	// 重新組合
	result := normalizedDate + timeStr

	// 3. 處理時間與時區 (Google API 必須要有時區)
	if timeStr != "" {
		// 4. 處理時區 (如果原始有時區，要轉成台北時間)
		if strings.Contains(dateStr, "Z") {
			// UTC 轉台北
			utcTime, _ := time.Parse(time.RFC3339, dateStr)
			taipeiTime := utcTime.In(time.FixedZone("Asia/Taipei", 8*60*60))
			return taipeiTime.Format("2006-01-02T15:04:05+08:00")
		}
		// 如果沒有包含時區資訊 (Z 或 +/-)，補上台北時區
		if !strings.ContainsAny(timeStr, "Z+-") {
			timeStr += "+08:00"
		}
		result = normalizedDate + timeStr
	}
	// 如果原始沒有時區，直接回傳修正後的日期+時間
	return result
}

// --- 核心修正：日曆 ID 智慧識別 ---
func resolveCalendarID(input string) string {
	cleanInput := strings.TrimSpace(input)
	lowInput := strings.ToLower(cleanInput)

	// 如果輸入為空或直接是 "primary"，直接回傳 primary
	if lowInput == "" || lowInput == "primary" {
		return "primary"
	}

	cals, err := getCalendarsJSON()
	if err != nil {
		return "primary" // 獲取失敗時的安全回退
	}

	// 1. 先嘗試精確 ID 匹配
	for _, c := range cals {
		if c.ID == cleanInput {
			return c.ID
		}
	}

	// 2. 再嘗試名稱模糊匹配 (不區分大小寫)
	for _, c := range cals {
		if strings.Contains(strings.ToLower(c.Summary), lowInput) {
			return c.ID
		}
	}

	// 3. 若都找不到，才報警告並回傳 primary
	fmt.Printf("⚠️  找不到匹配 [%s] 的日曆，將使用主日曆 (primary)\n", cleanInput)
	return "primary"
}

func main() {
	// 手動重組 os.Args：解決 cmd.exe 在不使用雙引號下將空白拆分成多個參數的問題
	var newArgs []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			newArgs = append(newArgs, arg)
		} else {
			if len(newArgs) > 0 {
				newArgs[len(newArgs)-1] += " " + arg
			} else {
				newArgs = append(newArgs, arg) // 避免第一個就是非 flag 的問題 (儘管理論上不會發生)
			}
		}
	}
	os.Args = append([]string{os.Args[0]}, newArgs...)

	// 參數定義
	today := time.Now().Format("2006-01-02")
	mode := flag.String("mode", "read", "功能模式: read 或 create")
	fromPtr := flag.String("from", today, "Start date (YYYY-MM-DD)")
	toPtr := flag.String("to", today, "End date (YYYY-MM-DD)")

	// Create 專用參數
	summaryPtr := flag.String("summary", "", "行程標題 (僅限 create 模式)")
	calIDPtr := flag.String("cal", "", "指定行事曆 ID (不填則使用預設)")
	rrulePtr := flag.String("rrule", "", "重複規則 (例如 RRULE:FREQ=MONTHLY;BYMONTHDAY=11)")
	reminders := flag.String("reminders", "", "提醒設定 (多個以逗號隔開, 例如: email:3d,popup:30m)")
	locationPtr := flag.String("location", "", "地點 (僅限 create 模式)")

	eventIDPtr := flag.String("event", "", "事件 ID (僅 delete 模式必填)") // 新增刪除專用參數

	forceStr := flag.String("force", "false", "是否忽略衝突強制執行") // 🌟 新增：強制執行標籤改為字串接收

	flag.Parse()

	// --- 1. 日期自動修正 (Normalization) ---
	// 處理如 2026-02-29 -> 2026-03-01 的自動進位
	normalizedFrom := normalizeDate(*fromPtr)
	normalizedTo := normalizeDate(*toPtr)

	if *mode == "delete" {
		runDeleteMode(normalizedFrom, normalizedTo, *summaryPtr, *calIDPtr, *eventIDPtr)
		return
	}
	if *mode == "create" {
		// --- 執行新增模式 ---
		if *summaryPtr == "" || normalizedFrom == "" || normalizedTo == "" {
			log.Fatal("❌ 新增行程時，summary, from, to 為必填項目")
		}

		targetID := *calIDPtr
		if targetID == "" {
			targetID = getDefaultCalendarID()
			fmt.Printf("ℹ️ 未指定 ID，自動選用主日曆: %s\n", targetID)
		} else {
			targetID = resolveCalendarID(targetID)
		}

		isForce := false
		if *forceStr == "true" || *forceStr == "1" {
			isForce = true
		}
		// 🌟 新增：建立前先檢查衝突
		if !isForce && hasConflict(targetID, normalizedFrom, normalizedTo, "") {
			os.Exit(1) // 發現衝突且未強制執行，則中斷並回傳非零退出碼
		}

		err := createEvent(targetID, *summaryPtr, normalizedFrom, normalizedTo, *rrulePtr, *reminders, *locationPtr)
		if err != nil {
			log.Fatalf("❌ 新增失敗: %v", err)
		}
		fmt.Println("✅ 行程新增成功！")

	} else if *mode == "update" {
		isForce := false
		if *forceStr == "true" || *forceStr == "1" {
			isForce = true
		}
		runUpdateMode(normalizedFrom, normalizedTo, *summaryPtr, *calIDPtr, *eventIDPtr, *rrulePtr, *locationPtr, isForce)
	} else {
		// --- 執行讀取模式 ---
		readCalendar(normalizedFrom, normalizedTo)
	}

}

// 輔助函式：判斷字串是否只包含英數字及連字號（Google Calendar ID 常見格式）
func isAlphanumericOrHyphen(s string) bool {
	for _, char := range s {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' {
			return false
		}
	}
	return true
}
