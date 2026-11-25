package model

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

var orderZhMap = map[string]string{
	"struthioniformes":   "鸵鸟目",
	"rheiformes":         "美洲鸵目",
	"tinamiformes":       "䳍形目",
	"casuariiformes":     "鹤鸵目",
	"apterygiformes":     "无翼鸟目",
	"anseriformes":       "雁形目",
	"galliformes":        "鸡形目",
	"phoenicopteriformes": "红鹳目",
	"podicipediformes":   "鸊鷉目",
	"columbiformes":      "鸽形目",
	"mesitornithiformes": "拟鹑目",
	"pterocliformes":     "沙鸡目",
	"otidiformes":        "鸨形目",
	"musophagiformes":    "蕉鹃目",
	"cuculiformes":       "鹃形目",
	"caprimulgiformes":   "夜鹰目",
	"apodiformes":        "雨燕目",
	"opisthocomiformes":  "麝雉目",
	"gruiformes":         "鹤形目",
	"charadriiformes":    "鸻形目",
	"eurypygiformes":     "日鳽目",
	"phaethontiformes":   "鹲形目",
	"gaviiformes":        "潜鸟目",
	"sphenisciformes":    "企鹅目",
	"procellariiformes":  "鹱形目",
	"ciconiiformes":      "鹳形目",
	"suliformes":         "鲣鸟目",
	"pelecaniformes":     "鹈形目",
	"cathartiformes":     "美洲鹫目",
	"accipitriformes":    "鹰形目",
	"strigiformes":       "鸮形目",
	"coliiformes":        "鼠鸟目",
	"leptosomiformes":    "鹃鴗目",
	"trogoniformes":      "咬鹃目",
	"bucerotiformes":     "犀鸟目",
	"coraciiformes":      "佛法僧目",
	"piciformes":         "啄木鸟目",
	"cariamiformes":      "叫鹤目",
	"falconiformes":      "隼形目",
	"psittaciformes":     "鹦形目",
	"passeriformes":      "雀形目",
}

var familySciZhMap = map[string]string{
	"anatidae":            "鸭科",
	"phasianidae":         "雉科",
	"phoenicopteridae":    "红鹳科",
	"podicipedidae":       "鸊鷉科",
	"columbidae":          "鸠鸽科",
	"pteroclidae":         "沙鸡科",
	"otididae":            "鸨科",
	"cuculidae":           "杜鹃科",
	"caprimulgidae":       "夜鹰科",
	"apodidae":            "雨燕科",
	"hemiprocnidae":       "凤头雨燕科",
	"rallidae":            "秧鸡科",
	"gruidae":             "鹤科",
	"burhinidae":          "石鸻科",
	"haematopodidae":      "蛎鹬科",
	"ibidorynchidae":      "鹮嘴鹬科",
	"ibidorhynchidae":     "鹮嘴鹬科",
	"recurvirostridae":    "反嘴鹬科",
	"charadriidae":        "鸻科",
	"rostratulidae":       "彩鹬科",
	"jacanidae":           "雉鸻科",
	"scolopacidae":        "鹬科",
	"turnicidae":          "三趾鹑科",
	"glareolidae":         "燕鸻科",
	"sterocorariidae":     "贼鸥科",
	"stercorariidae":      "贼鸥科",
	"alcidae":             "海雀科",
	"laridae":             "鸥科",
	"phaethontidae":       "鹲科",
	"gaviidae":            "潜鸟科",
	"diomedeidae":         "信天翁科",
	"hydrobatidae":        "海燕科",
	"procellariidae":      "鹱科",
	"ciconiidae":          "鹳科",
	"fregatidae":          "军舰鸟科",
	"sulidae":             "鲣鸟科",
	"anhingidae":          "蛇鹈科",
	"phalacrocoracidae":   "鸬鹚科",
	"pelecanidae":         "鹈鹕科",
	"ardeidae":            "鹭科",
	"threskiornithidae":   "鹮科",
	"pandionidae":         "鹗科",
	"accipitridae":        "鹰科",
	"tytonidae":           "草鸮科",
	"strigidae":           "鸱鸮科",
	"trogonidae":          "咬鹃科",
	"bucerotidae":         "犀鸟科",
	"upupidae":            "戴胜科",
	"alcedinidae":         "翠鸟科",
	"meropidae":           "蜂虎科",
	"coraciidae":          "佛法僧科",
	"megalaimidae":        "拟啄木鸟科",
	"picidae":             "啄木鸟科",
	"falconidae":          "隼科",
	"psittaculidae":       "长尾鹦鹉科",
	"psittacidae":         "鹦鹉科",
	"eurylaimidae":        "阔嘴鸟科",
	"pittidae":            "八色鸫科",
	"campephagidae":       "山椒鸟科",
	"pachycephalidae":     "啸鹟科",
	"oriolidae":           "黄鹂科",
	"artamidae":           "燕鵙科",
	"vireonidae":          "绿鹃科",
	"vangidae":            "钩嘴鵙科",
	"dicruridae":          "卷尾科",
	"rhipiduridae":        "扇尾鹟科",
	"laniidae":            "伯劳科",
	"corvidae":            "鸦科",
	"monarchidae":         "王鹟科",
	"stenostiridae":       "仙鹟科",
	"paridae":             "山雀科",
	"remizidae":           "攀雀科",
	"alaudidae":           "百灵科",
	"panuridae":           "文须雀科",
	"cisticolidae":        "扇尾莺科",
	"acrocephalidae":      "苇莺科",
	"locustellidae":       "蝗莺科",
	"hirundinidae":        "燕科",
	"pycnonotidae":        "鹎科",
	"phylloscopidae":      "柳莺科",
	"cettiidae":           "树莺科",
	"aegithalidae":        "长尾山雀科",
	"sylviidae":           "莺科",
	"paradoxornithidae":   "鸦雀科",
	"zosteropidae":        "绣眼鸟科",
	"timaliidae":          "画眉科",
	"pellorneidae":        "幽鹛科",
	"leiothrichidae":      "噪鹛科",
	"regulidae":           "戴菊科",
	"elachuridae":         "丽星鹩鹛科",
	"tichodromidae":       "旋壁雀科",
	"sittidae":            "䴓科",
	"certhiidae":          "旋木雀科",
	"troglodytidae":       "鹪鹩科",
	"cinclidae":           "河乌科",
	"sturnidae":           "椋鸟科",
	"turdidae":            "鸫科",
	"muscicapidae":        "鹟科",
	"bombycillidae":       "太平鸟科",
	"hypocoliidae":        "连雀科",
	"dicaeidae":           "啄花鸟科",
	"nectariniidae":       "太阳鸟科",
	"irenidae":            "和平鸟科",
	"chloropseidae":       "叶鹎科",
	"prunellidae":         "岩鹨科",
	"ploceidae":           "织布鸟科",
	"estrildidae":         "梅花雀科",
	"passeridae":          "雀科",
	"motacillidae":        "鹡鸰科",
	"fringillidae":        "燕雀科",
	"calcariidae":         "铁爪鹀科",
	"emberizidae":         "鹀科",
	"cardinalidae":        "美洲雀科",
	"icteridae":           "拟鹂科",
	"parulidae":           "森莺科",
	"spheniscidae":        "企鹅科",
	"trochilidae":         "蜂鸟科",
	"tyrannidae":          "霸鹟科",
}

var familyComZhMap = map[string]string{
	"ducks, geese, and waterfowl":              "鸭科",
	"pheasants, grouse, and allies":            "雉科",
	"flamingos":                                "红鹳科",
	"grebes":                                   "鸊鷉科",
	"pigeons and doves":                        "鸠鸽科",
	"sandgrouse":                               "沙鸡科",
	"bustards":                                 "鸨科",
	"cuckoos":                                  "杜鹃科",
	"nightjars and allies":                     "夜鹰科",
	"swifts":                                   "雨燕科",
	"rails, gallinules, and coots":             "秧鸡科",
	"cranes":                                   "鹤科",
	"thick-knees":                              "石鸻科",
	"oystercatchers":                           "蛎鹬科",
	"ibisbill":                                 "鹮嘴鹬科",
	"stilts and avocets":                       "反嘴鹬科",
	"plovers and lapwings":                     "鸻科",
	"painted-snipes":                           "彩鹬科",
	"jacanas":                                  "雉鸻科",
	"sandpipers and allies":                    "鹬科",
	"buttonquail":                              "三趾鹑科",
	"pratincoles and coursers":                 "燕鸻科",
	"skuas and jaegers":                        "贼鸥科",
	"auks, murres, and puffins":                "海雀科",
	"gulls, terns, and skimmers":               "鸥科",
	"tropicbirds":                              "鹲科",
	"loons":                                    "潜鸟科",
	"albatrosses":                              "信天翁科",
	"northern storm-petrels":                   "海燕科",
	"shearwaters and petrels":                  "鹱科",
	"storks":                                   "鹳科",
	"frigatebirds":                             "军舰鸟科",
	"boobies and gannets":                      "鲣鸟科",
	"darters":                                  "蛇鹈科",
	"cormorants and shags":                     "鸬鹚科",
	"pelicans":                                 "鹈鹕科",
	"herons, egrets, and bitterns":             "鹭科",
	"ibises and spoonbills":                    "鹮科",
	"osprey":                                   "鹗科",
	"hawks, eagles, and kites":                 "鹰科",
	"barn-owls":                                "草鸮科",
	"owls":                                     "鸱鸮科",
	"trogons":                                  "咬鹃科",
	"hornbills":                                "犀鸟科",
	"hoopoes":                                  "戴胜科",
	"kingfishers":                              "翠鸟科",
	"bee-eaters":                               "蜂虎科",
	"rollers":                                  "佛法僧科",
	"asian barbets":                            "拟啄木鸟科",
	"woodpeckers":                              "啄木鸟科",
	"falcons and caracaras":                    "隼科",
	"old world parrots":                        "长尾鹦鹉科",
	"pittas":                                   "八色鸫科",
	"cuckooshrikes":                            "山椒鸟科",
	"whistlers and allies":                     "啸鹟科",
	"old world orioles":                        "黄鹂科",
	"woodswallows, bellmagpies, and allies":    "燕鵙科",
	"vireos, shrike-babblers, and erpornis":    "绿鹃科",
	"vangas, helmetshrikes, and allies":        "钩嘴鵙科",
	"drongos":                                  "卷尾科",
	"fantails":                                 "扇尾鹟科",
	"shrikes":                                  "伯劳科",
	"crows, jays, and magpies":                 "鸦科",
	"monarch flycatchers":                      "王鹟科",
	"fairy flycatchers":                        "仙鹟科",
	"tits, chickadees, and titmice":            "山雀科",
	"penduline-tits":                           "攀雀科",
	"larks":                                    "百灵科",
	"bearded reedling":                         "文须雀科",
	"cisticolas and allies":                    "扇尾莺科",
	"reed warblers and allies":                 "苇莺科",
	"grassbirds and allies":                    "蝗莺科",
	"swallows":                                 "燕科",
	"bulbuls":                                  "鹎科",
	"leaf warblers":                            "柳莺科",
	"bush warblers and allies":                 "树莺科",
	"long-tailed tits":                         "长尾山雀科",
	"sylviid warblers, parrotbills, and allies": "莺科",
	"parrotbills":                              "鸦雀科",
	"white-eyes, yuhinas, and allies":          "绣眼鸟科",
	"tree-babblers, scimitar-babblers, and allies": "画眉科",
	"ground babblers and allies":               "幽鹛科",
	"laughingthrushes and allies":              "噪鹛科",
	"kinglets":                                 "戴菊科",
	"wallcreeper":                              "旋壁雀科",
	"nuthatches":                               "䴓科",
	"treecreepers":                             "旋木雀科",
	"wrens":                                    "鹪鹩科",
	"dippers":                                  "河乌科",
	"starlings":                                "椋鸟科",
	"thrushes and allies":                      "鸫科",
	"old world flycatchers":                    "鹟科",
	"waxwings":                                 "太平鸟科",
	"flowerpeckers":                            "啄花鸟科",
	"sunbirds and spiderhunters":               "太阳鸟科",
	"fairy-bluebirds":                          "和平鸟科",
	"leafbirds":                                "叶鹎科",
	"accentors":                                "岩鹨科",
	"weavers and allies":                       "织布鸟科",
	"waxbills and allies":                      "梅花雀科",
	"old world sparrows":                       "雀科",
	"wagtails and pipits":                      "鹡鸰科",
	"finches, euphonias, and allies":           "燕雀科",
	"longspurs and snow buntings":              "铁爪鹀科",
	"old world buntings":                       "鹀科",
}

func containsCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// OrderZh 将鸟类目的学名转换为简体中文名称。
func OrderZh(order string) string {
	trimmed := strings.TrimSpace(order)
	if trimmed == "" {
		return ""
	}
	if containsCJK(trimmed) {
		return trimmed
	}
	if zh, ok := orderZhMap[strings.ToLower(trimmed)]; ok {
		return zh
	}
	return trimmed
}

// FamilyZh 优先将鸟类科名（拉丁科名或英文俗名）转换为简体中文名称。
func FamilyZh(familySci, familyCom string) string {
	com := strings.TrimSpace(familyCom)
	sci := strings.TrimSpace(familySci)
	if containsCJK(com) {
		return com
	}
	if sci != "" {
		if zh, ok := familySciZhMap[strings.ToLower(sci)]; ok {
			return zh
		}
	}
	if com != "" {
		if zh, ok := familyComZhMap[strings.ToLower(com)]; ok {
			return zh
		}
	}
	if com != "" {
		return com
	}
	return sci
}

// FormatTaxonLine 将鸟种分类信息格式化为纯中文“目 · 科”标签。
func FormatTaxonLine(t Taxon) string {
	ord := OrderZh(t.Order)
	fam := FamilyZh(t.FamilySciName, t.FamilyComName)
	switch {
	case ord != "" && fam != "":
		return ord + " · " + fam
	case fam != "":
		return fam
	default:
		return ord
	}
}

// QualityGradeZh 将 iNaturalist 观测质量等级转为中文标签。
func QualityGradeZh(grade string) string {
	switch strings.ToLower(strings.TrimSpace(grade)) {
	case "research":
		return "研究级"
	case "needs_id":
		return "待复核"
	case "casual":
		return "休闲记录"
	default:
		if strings.TrimSpace(grade) == "" {
			return "现场实拍"
		}
		return grade
	}
}

// FormatObsTime 将 eBird / iNaturalist 的原始时间字符串转换为友好的简体中文格式。
func FormatObsTime(raw string) string {
	return formatObsTimeAt(raw, time.Now())
}

func formatObsTimeAt(raw string, now time.Time) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "近期记录"
	}

	layoutsWithTime := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04",
	}
	for _, layout := range layoutsWithTime {
		if tm, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return formatDateAndClock(tm, now, true)
		}
	}

	layoutsDateOnly := []string{
		"2006-01-02",
		"2006/01/02",
	}
	for _, layout := range layoutsDateOnly {
		if tm, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return formatDateAndClock(tm, now, false)
		}
	}
	return s
}

func formatDateAndClock(tm, now time.Time, hasClock bool) string {
	y1, m1, d1 := tm.Date()
	y2, m2, d2 := now.Date()
	clock := ""
	if hasClock {
		clock = " " + tm.Format("15:04")
	}

	today := time.Date(y2, m2, d2, 0, 0, 0, 0, now.Location())
	targetDay := time.Date(y1, m1, d1, 0, 0, 0, 0, now.Location())
	diffDays := int(today.Sub(targetDay).Hours() / 24)

	switch diffDays {
	case 0:
		if hasClock {
			return "今天" + clock
		}
		return "今天"
	case 1:
		return "昨天" + clock
	default:
		if y1 == y2 {
			return fmt.Sprintf("%d月%d日%s", int(m1), d1, clock)
		}
		return fmt.Sprintf("%d年%d月%d日%s", y1, int(m1), d1, clock)
	}
}
