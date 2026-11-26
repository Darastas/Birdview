package model

import (
	"fmt"
	"strings"
	"time"
)

// SeasonalPhase 表示某鸟种在特定地区与月份的时令迁徙状态。
type SeasonalPhase int

const (
	PhaseAbsent       SeasonalPhase = iota
	PhaseLeaving                    // 马上离开（迁离末期，把握最后窗口）
	PhaseInSeason                   // 当季（主力停歇 / 越冬 / 繁殖高峰）
	PhaseJustArrived                // 刚到（先头部队初抵）
	PhaseArrivingSoon               // 马上到（下月将抵达，适合提前规划）
)

func (p SeasonalPhase) Label() string {
	switch p {
	case PhaseLeaving:
		return "马上离开"
	case PhaseInSeason:
		return "当季"
	case PhaseJustArrived:
		return "刚到"
	case PhaseArrivingSoon:
		return "马上到"
	default:
		return "非本季"
	}
}

func (p SeasonalPhase) ShortActionHint() string {
	switch p {
	case PhaseLeaving:
		return "迁离窗口尾声，抓紧最后几天"
	case PhaseInSeason:
		return "当前正值最佳观赏期，最易与它邂逅"
	case PhaseJustArrived:
		return "本月先头部队刚抵境，抢先打卡"
	case PhaseArrivingSoon:
		return "下月即将抵境，建议提前定日历"
	default:
		return "暂不在该地区活跃窗口"
	}
}

// MonthZh 返回月份对应的中文文本（如 "十一月"）。
func MonthZh(month int) string {
	names := [...]string{
		"", "一月", "二月", "三月", "四月", "五月", "六月",
		"七月", "八月", "九月", "十月", "十一月", "十二月",
	}
	if month < 1 || month >= len(names) {
		return ""
	}
	return names[month]
}

// EcoZone 将中国 34 个省级行政区按鸟类迁徙地理生态带划分为 7 大区域（外加全国汇总）。
type EcoZone string

const (
	ZoneNorth     EcoZone = "north"     // 华北（京津冀晋蒙鲁）
	ZoneNortheast EcoZone = "northeast" // 东北（黑吉辽）
	ZoneEast      EcoZone = "east"      // 华东（沪苏浙皖闽赣台）
	ZoneCentral   EcoZone = "central"   // 华中（豫鄂湘）
	ZoneSouth     EcoZone = "south"     // 华南（粤桂琼港澳）
	ZoneSouthwest EcoZone = "southwest" // 西南（川渝云贵藏）
	ZoneNorthwest EcoZone = "northwest" // 西北（陕甘青宁新）
	ZoneNational  EcoZone = "national"  // 中国全国
)

func EcoZoneLabel(z EcoZone) string {
	switch z {
	case ZoneNorth:
		return "华北"
	case ZoneNortheast:
		return "东北"
	case ZoneEast:
		return "华东"
	case ZoneCentral:
		return "华中"
	case ZoneSouth:
		return "华南"
	case ZoneSouthwest:
		return "西南"
	case ZoneNorthwest:
		return "西北"
	default:
		return "全国"
	}
}

// RegionCodeToEcoZone 将 eBird 省级代码映射至对应的中国生态地理分区。
func RegionCodeToEcoZone(code string) EcoZone {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "CN-11", "CN-12", "CN-13", "CN-14", "CN-15", "CN-37":
		return ZoneNorth
	case "CN-21", "CN-22", "CN-23":
		return ZoneNortheast
	case "CN-31", "CN-32", "CN-33", "CN-34", "CN-35", "CN-36", "TW":
		return ZoneEast
	case "CN-41", "CN-42", "CN-43":
		return ZoneCentral
	case "CN-44", "CN-45", "CN-46", "HK", "MO":
		return ZoneSouth
	case "CN-50", "CN-51", "CN-52", "CN-53", "CN-54":
		return ZoneSouthwest
	case "CN-61", "CN-62", "CN-63", "CN-64", "CN-65":
		return ZoneNorthwest
	default:
		return ZoneNational
	}
}

// MonthSeasonTheme 返回指定月份在中国观鸟日历中的自然时令概述。
func MonthSeasonTheme(month int) string {
	switch month {
	case 1:
		return "隆冬越冬盛期 · 湿地水鸟与林间冬候集群"
	case 2:
		return "早春北迁前奏 · 首批雁鸭与鹤类开始躁动"
	case 3:
		return "春迁启幕 · 鸻鹬先头北上，冬候鸟陆续告别"
	case 4:
		return "春季迁徙高峰 · 柳莺、歌鸲与猛禽过境潮"
	case 5:
		return "晚春繁殖换羽 · 繁殖羽最艳丽，林鸟鸣唱热烈"
	case 6:
		return "仲夏育雏期 · 夏候鸟筑巢育雏，高山林鸟活跃"
	case 7:
		return "盛夏留守与初离 · 幼鸟离巢，涉禽开启早秋南返"
	case 8:
		return "早秋南迁启动 · 鸻鹬与雨燕率先踏上南归路"
	case 9:
		return "秋迁猛禽与林鸟潮 · 过境小鸟换上秋羽"
	case 10:
		return "深秋大迁徙 · 鹤鹳雁鸭南下，冬候鸟陆续报到"
	case 11:
		return "初冬越冬集结 · 越冬水鸟大批进驻南方与沿海湿地"
	case 12:
		return "寒冬稳定越冬 · 南北越冬地最易近距离观察大群"
	default:
		return "中国四季迁徙观鸟时令"
	}
}

// SeasonalBird 描述一条经核验的中国观鸟时令指南物种档案（聚焦迁徙时间、地点与羽色变化）。
type SeasonalBird struct {
	SpeciesCode       string
	ComName           string
	SciName           string
	Order             string
	FamilySciName     string
	DefaultMonths     [12]SeasonalPhase
	ZoneMonths        map[EcoZone][12]SeasonalPhase
	StrictZones       bool // 为 true 时仅在 ZoneMonths 明确列出的生态大区出现（全国视图仍按 DefaultMonths 展示）
	SummaryNote       string
	LeavingTip        string
	InSeasonTip       string
	JustArrivedTip    string
	ArrivingSoonTip   string
	Habitats          string
	BestTimeOfDay     string
	ZoneSpots         map[EcoZone]string
	NationalSpots     string
	MigrationTimeline string
	PlumageChanges    string
	BehaviorChanges   string
	FieldTips         string
}

// PhaseFor 返回该物种在指定生态区与月份（1-12）的物候状态。
func (b SeasonalBird) PhaseFor(zone EcoZone, month int) SeasonalPhase {
	if month < 1 || month > 12 {
		return PhaseAbsent
	}
	idx := month - 1
	if zone == ZoneNational {
		if b.DefaultMonths[idx] != PhaseAbsent {
			return b.DefaultMonths[idx]
		}
		for _, arr := range b.ZoneMonths {
			if arr[idx] != PhaseAbsent {
				return arr[idx]
			}
		}
		return PhaseAbsent
	}
	if arr, ok := b.ZoneMonths[zone]; ok {
		return arr[idx]
	}
	if b.StrictZones {
		return PhaseAbsent
	}
	return b.DefaultMonths[idx]
}

// MonthCalendar 返回该物种在指定生态区全年 1-12 月的物候数组。
func (b SeasonalBird) MonthCalendar(zone EcoZone) [12]SeasonalPhase {
	if zone != ZoneNational {
		if arr, ok := b.ZoneMonths[zone]; ok {
			return arr
		}
	}
	return b.DefaultMonths
}

// RecommendedSpots 返回贴合当前生态区的推荐观鸟点位描述。
func (b SeasonalBird) RecommendedSpots(zone EcoZone) string {
	if zone != ZoneNational {
		if s, ok := b.ZoneSpots[zone]; ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return b.NationalSpots
}

// CardSeasonBlurb 返回卡片上针对当前月份与状态的简短时令看点（1-2 句）。
func (b SeasonalBird) CardSeasonBlurb(phase SeasonalPhase, month int, regionName string) string {
	switch phase {
	case PhaseLeaving:
		if b.LeavingTip != "" {
			return b.LeavingTip
		}
	case PhaseInSeason:
		if b.InSeasonTip != "" {
			return b.InSeasonTip
		}
	case PhaseJustArrived:
		if b.JustArrivedTip != "" {
			return b.JustArrivedTip
		}
	case PhaseArrivingSoon:
		if b.ArrivingSoonTip != "" {
			return b.ArrivingSoonTip
		}
	}
	if b.SummaryNote != "" {
		return b.SummaryNote
	}
	return fmt.Sprintf("%d 月在%s处于%s阶段，建议前往%s寻找。", month, regionName, phase.Label(), b.Habitats)
}

// BirdingMemo 表示用户的追鸟备忘/观鸟行程计划（支持同步日历与图片分享）。
type BirdingMemo struct {
	ID          string    `json:"id"`
	SpeciesCode string    `json:"speciesCode"`
	ComName     string    `json:"comName"`
	SciName     string    `json:"sciName"`
	RegionName  string    `json:"regionName"`
	TargetMonth int       `json:"targetMonth"`
	TargetDate  string    `json:"targetDate"` // YYYY-MM-DD
	PhaseLabel  string    `json:"phaseLabel"`
	Location    string    `json:"location"`
	Notes       string    `json:"notes"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
}
