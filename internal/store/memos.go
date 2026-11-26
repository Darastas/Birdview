package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/model"
)

func (s *Store) memosPath() string { return filepath.Join(s.dir, "memos.json") }

// Memos 返回全部追鸟备忘，未完成项按目标日期升序排在前方，已完成项排在后方。
func (s *Store) Memos() []model.BirdingMemo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.readMemosLocked()
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Completed != list[j].Completed {
			return !list[i].Completed
		}
		if list[i].TargetDate != list[j].TargetDate {
			return list[i].TargetDate < list[j].TargetDate
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list
}

func (s *Store) readMemosLocked() []model.BirdingMemo {
	data, err := os.ReadFile(s.memosPath())
	if err != nil {
		return nil
	}
	var out []model.BirdingMemo
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

// AddMemo 新增一条追鸟备忘并持久化。
func (s *Store) AddMemo(m model.BirdingMemo) (model.BirdingMemo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m.ID == "" {
		var b [6]byte
		_, _ = rand.Read(b[:])
		m.ID = "memo_" + hex.EncodeToString(b[:])
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	if m.TargetDate == "" {
		m.TargetDate = defaultTargetDateForMonth(m.TargetMonth)
	}
	list := s.readMemosLocked()
	list = append([]model.BirdingMemo{m}, list...)
	if err := writeJSON(s.memosPath(), list); err != nil {
		return m, err
	}
	return m, nil
}

// ToggleMemoDone 切换追鸟备忘的完成/待追状态。
func (s *Store) ToggleMemoDone(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.readMemosLocked()
	changed := false
	for i := range list {
		if list[i].ID == id {
			list[i].Completed = !list[i].Completed
			changed = true
			break
		}
	}
	if !changed {
		return nil
	}
	return writeJSON(s.memosPath(), list)
}

// DeleteMemo 删除一条追鸟备忘。
func (s *Store) DeleteMemo(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.readMemosLocked()
	out := make([]model.BirdingMemo, 0, len(list))
	for _, m := range list {
		if m.ID != id {
			out = append(out, m)
		}
	}
	return writeJSON(s.memosPath(), out)
}

// HasActiveMemo 检查指定鸟种在指定月份是否已有未完成的追鸟备忘。
func (s *Store) HasActiveMemo(speciesCode string, month int) bool {
	for _, m := range s.Memos() {
		if !m.Completed && m.SpeciesCode == speciesCode && (month == 0 || m.TargetMonth == month) {
			return true
		}
	}
	return false
}

func defaultTargetDateForMonth(month int) string {
	now := time.Now()
	if month < 1 || month > 12 {
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	}
	year := now.Year()
	day := now.Day()
	if month < int(now.Month()) {
		year++
		day = 10
	} else if month > int(now.Month()) {
		day = 8
	} else {
		// 当月默认设为最近的周六或次日
		cand := now.AddDate(0, 0, 1)
		for i := 0; i < 7; i++ {
			d := now.AddDate(0, 0, 1+i)
			if d.Weekday() == time.Saturday && int(d.Month()) == month {
				cand = d
				break
			}
		}
		return cand.Format("2006-01-02")
	}
	return time.Date(year, time.Month(month), day, 7, 30, 0, 0, time.Local).Format("2006-01-02")
}

// BuildCalendarICS 生成标准 RFC 5545 日历日程文件（含提前提醒），兼容 iOS/Android/macOS/Windows 日历导入。
func BuildCalendarICS(title, location, desc string, start time.Time, duration time.Duration) string {
	if duration <= 0 {
		duration = 2 * time.Hour
	}
	end := start.Add(duration)
	esc := func(v string) string {
		r := strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, "\r\n", `\n`, "\n", `\n`)
		return r.Replace(strings.TrimSpace(v))
	}
	uid := fmt.Sprintf("birdview-%d@birdview.app", start.UnixNano())
	nowUTC := time.Now().UTC().Format("20060102T150405Z")
	dtStart := start.UTC().Format("20060102T150405Z")
	dtEnd := end.UTC().Format("20060102T150405Z")

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Birdview//Seasonal Birding Guide//CN\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + uid + "\r\n")
	b.WriteString("DTSTAMP:" + nowUTC + "\r\n")
	b.WriteString("DTSTART:" + dtStart + "\r\n")
	b.WriteString("DTEND:" + dtEnd + "\r\n")
	b.WriteString("SUMMARY:" + esc(title) + "\r\n")
	if strings.TrimSpace(location) != "" {
		b.WriteString("LOCATION:" + esc(location) + "\r\n")
	}
	if strings.TrimSpace(desc) != "" {
		b.WriteString("DESCRIPTION:" + esc(desc) + "\r\n")
	}
	b.WriteString("BEGIN:VALARM\r\n")
	b.WriteString("ACTION:DISPLAY\r\n")
	b.WriteString("DESCRIPTION:" + esc("观鸟日程提醒："+title) + "\r\n")
	b.WriteString("TRIGGER:-PT12H\r\n")
	b.WriteString("END:VALARM\r\n")
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}
