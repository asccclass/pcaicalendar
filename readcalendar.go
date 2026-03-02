package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func getEventsJSON(calendarID, from, to string) ([]Event, error) {
	// 如果 from 與 to 相同且只是日期（不含 T），將 to 向後推一天以包含整天
	if from == to && !strings.Contains(to, "T") {
		if tEnd, err := time.Parse("2006-01-02", to); err == nil {
			to = tEnd.AddDate(0, 0, 1).Format("2006-01-02")
		}
	}

	output, err := exec.Command(getGogPath(), "calendar", "events", calendarID,
		"--from", from, "--to", to, "--json").CombinedOutput()
	if err != nil {
		return nil, err
	}
	outStr := string(output)
	if strings.Contains(strings.ToLower(outStr), "error") || strings.Contains(outStr, "400") {
		fmt.Printf("⚠️  指令看似成功執行，但 Google 回傳了錯誤：\n%s\n", outStr)
	}
	var resp EventResponse
	json.Unmarshal(output, &resp)
	return resp.Events, nil
}

func getCalendarsJSON() ([]Calendar, error) {
	output, err := exec.Command(getGogPath(), "calendar", "calendars", "--json").CombinedOutput()
	if err != nil {
		return nil, err
	}
	var resp CalendarListResponse
	json.Unmarshal(output, &resp)
	return resp.Calendars, nil
}

func readCalendar(fromPtr, toPtr string) {
	realToDate := toPtr

	// 2. 取得日曆清單
	allCalendars, err := getCalendarsJSON()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	var ownerCalendars []Calendar
	for _, cal := range allCalendars {
		if cal.AccessRole == "owner" {
			ownerCalendars = append(ownerCalendars, cal)
		}
	}

	// 3. 並行抓取
	var wg sync.WaitGroup
	resultsChan := make(chan FinalOutput, len(ownerCalendars))

	for _, cal := range ownerCalendars {
		wg.Add(1)
		go func(c Calendar) {
			defer wg.Done()
			events, err := getEventsJSON(c.ID, fromPtr, realToDate)
			if err != nil || len(events) == 0 {
				return
			}

			var formattedEvents []FormattedEvent
			for _, e := range events {
				startStr, endStr := processEventTimes(e)

				formattedEvents = append(formattedEvents, FormattedEvent{
					ID:        e.ID,
					StartTime: startStr,
					EndTime:   endStr,
					EventName: e.Summary,
					Location:  e.Location,    // 填入地點
					Summary:   e.Description, // 描述放在 summary
				})
			}

			resultsChan <- FinalOutput{
				Creator: c.Summary,
				Events:  formattedEvents,
			}
		}(cal)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var finalResults []FinalOutput
	for res := range resultsChan {
		finalResults = append(finalResults, res)
	}

	if len(finalResults) == 0 {
		fmt.Println("[]")
		return
	}

	jsonBytes, _ := json.MarshalIndent(finalResults, "", "  ")
	fmt.Println(string(jsonBytes))
}
