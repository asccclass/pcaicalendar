# PCAI Calendar

## 專案說明
這是一個基於 Go 語言開發的命令列日曆管理工具。專案透過封裝並調用 `gog` (Google API 終端機工具) 來與 Google Calendar 進行互動。
此工具提供更友善且智慧化的操作介面，支援跨日曆存取、名稱模糊比對、防呆衝突檢查，以及進階的時間與時區處理，適合用於排程任務或者快速讀寫行事曆。

## 主要功能
- **讀取行程 (`read`):** 讀取具有 Owner 權限的日曆行程，處理跨日與全天的時間顯示，將行程整合並以 JSON 陣列格式輸出。
- **新增行程 (`create`):** 快速新增行程，可設定標題、時間、地點，並支援重複規則 (`--rrule`) 與多種提醒方式 (`--reminders`)。新增前會自動偵測並阻擋時段衝突，可透過 (`--force`) 強制新增。
- **更新行程 (`update`):** 利用精確的事件 ID 或者透過模糊搜尋原標題找到目標行程，快速變更時間、地點與其他設定。
- **刪除行程 (`delete`):** 可透過精確 ID 或事件標題智慧比對，自動尋找目標行程後進行刪除操作。
- **智慧日曆識別:** 只需要輸入日曆名稱的部分關鍵字即可自動匹配正確的 ID，若未指定則自動選擇主要（primary）日曆。
- **智慧日期時間處理:** 內建時間進位與修正機制 (例如解決 2/29 溢出問題)，會自動加上區域時區（+08:00）。

## 使用方式
程式編譯後的執行檔為 `calendar.exe` (Windows) 或 `calendar` (Linux/macOS) (位於 `bin/` 目錄或與程式同層級)。

### 基本語法
```bash
calendar.exe --mode <功能模式> [參數...]
```
若未提供 `--mode` 參數，則預設為 `read`。

### 支援參數
| 參數名稱 | 說明 | 適用模式 |
| -------- | ---- | -------- |
| `--mode` | 功能模式設定，可為 `read`, `create`, `update`, `delete` | 所有 (預設 `read`) |
| `--from` | 開始日期，支援 `YYYY-MM-DD` 或 RFC3339 日期時間 | 所有 (預設為今天) |
| `--to` | 結束日期，支援 `YYYY-MM-DD` 或 RFC3339 日期時間 | 所有 (預設為今天) |
| `--summary` | 行程標題，若沒有 ID 時可作為搜尋匹配用 | `create`, `update`, `delete` |
| `--cal` | 指定目標行事曆的 ID 或是模糊名稱字串 | 所有 |
| `--rrule` | 設定重複規則 (例如 `RRULE:FREQ=MONTHLY;BYMONTHDAY=11`) | `create`, `update` |
| `--reminders` | 設定自動提醒設定 (多個以逗號隔開, 例如: `email:3d,popup:30m`) | `create` |
| `--location` | 指定或更新行程地點 | `create`, `update` |
| `--event` | 提供明確的 Google Calendar 事件 ID | `update`, `delete` |
| `--force` | 忽略時段衝突而強制執行建立或更新的情境 (布林值預設 `false`) | `create`, `update` |

### 執行範例
**1. 讀取今天的行程 (預設模式):**
```bash
calendar.exe
```

**2. 讀取特定日期區間的行程:**
```bash
calendar.exe --from 2026-02-27 --to 2026-02-27 
```

**3. 新增行程:**
```bash
calendar.exe --mode create --summary "部門會議" --from 2026-03-05T10:00:00 --to 2026-03-05T11:00:00 --location "會議室 A"
```

**4. 發現衝突時強制新增行程:**
```bash
calendar.exe --mode create --summary "臨時討論" --from 2026-03-05T10:00:00 --to 2026-03-05T11:00:00 --force
```

**5. 透過標題名稱搜尋並更新行程:**
```bash
calendar.exe --mode update --summary "部門會議" --location "會議室 B"
```

**6. 透過標題名稱搜尋並刪除行程:**
```bash
calendar.exe --mode delete --summary "部門會議" --from 2026-03-05 --to 2026-03-05
```