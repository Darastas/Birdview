package model

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

var orderZhMap = map[string]string{
	"struthioniformes":    "鸵鸟目",
	"rheiformes":          "美洲鸵目",
	"tinamiformes":        "䳍形目",
	"casuariiformes":      "鹤鸵目",
	"apterygiformes":      "无翼鸟目",
	"anseriformes":        "雁形目",
	"galliformes":         "鸡形目",
	"phoenicopteriformes": "红鹳目",
	"podicipediformes":    "鸊鷉目",
	"columbiformes":       "鸽形目",
	"mesitornithiformes":  "拟鹑目",
	"pterocliformes":      "沙鸡目",
	"otidiformes":         "鸨形目",
	"musophagiformes":     "蕉鹃目",
	"cuculiformes":        "鹃形目",
	"caprimulgiformes":    "夜鹰目",
	"steatornithiformes":  "油鸱目",
	"nyctibiiformes":      "林鸱目",
	"podargiformes":       "蛙口夜鹰目",
	"aegotheliformes":     "裸鼻鸱目",
	"apodiformes":         "雨燕目",
	"opisthocomiformes":   "麝雉目",
	"gruiformes":          "鹤形目",
	"charadriiformes":     "鸻形目",
	"eurypygiformes":      "日鳽目",
	"phaethontiformes":    "鹲形目",
	"gaviiformes":         "潜鸟目",
	"sphenisciformes":     "企鹅目",
	"procellariiformes":   "鹱形目",
	"ciconiiformes":       "鹳形目",
	"suliformes":          "鲣鸟目",
	"pelecaniformes":      "鹈形目",
	"cathartiformes":      "美洲鹫目",
	"accipitriformes":     "鹰形目",
	"strigiformes":        "鸮形目",
	"coliiformes":         "鼠鸟目",
	"leptosomiformes":     "鹃鴗目",
	"trogoniformes":       "咬鹃目",
	"bucerotiformes":      "犀鸟目",
	"coraciiformes":       "佛法僧目",
	"piciformes":          "啄木鸟目",
	"cariamiformes":       "叫鹤目",
	"falconiformes":       "隼形目",
	"psittaciformes":      "鹦形目",
	"passeriformes":       "雀形目",
}

var familySciZhMap = map[string]string{
	"acanthisittidae":    "刺鹩科",
	"acanthizidae":       "细嘴莺科",
	"accipitridae":       "鹰科",
	"acrocephalidae":     "苇莺科",
	"aegithalidae":       "长尾山雀科",
	"aegithinidae":       "雀鹎科",
	"aegothelidae":       "裸鼻鸱科",
	"alaudidae":          "百灵科",
	"alcedinidae":        "翠鸟科",
	"alcidae":            "海雀科",
	"anatidae":           "鸭科",
	"anhimidae":          "叫鸭科",
	"anhingidae":         "蛇鹈科",
	"anseranatidae":      "鹊鹅科",
	"apodidae":           "雨燕科",
	"apterygidae":        "几维科",
	"aramidae":           "秧鹤科",
	"ardeidae":           "鹭科",
	"artamidae":          "燕鵙科",
	"atrichornithidae":   "薮鸟科",
	"balaenicipitidae":   "鲸头鹳科",
	"bernieridae":        "马岛莺科",
	"bombycillidae":      "太平鸟科",
	"brachypteraciidae":  "地佛法僧科",
	"bucconidae":         "喷鴷科",
	"bucerotidae":        "犀鸟科",
	"buphagidae":         "牛椋鸟科",
	"burhinidae":         "石鸻科",
	"cacatuidae":         "凤头鹦鹉科",
	"calcariidae":        "铁爪鹀科",
	"callaeidae":         "垂耳鸦科",
	"calyptomenidae":     "绿阔嘴鸟科",
	"calyptophilidae":    "聊唐纳雀科",
	"campephagidae":      "山椒鸟科",
	"capitonidae":        "须鴷科",
	"caprimulgidae":      "夜鹰科",
	"cardinalidae":       "美洲雀科",
	"cariamidae":         "叫鹤科",
	"casuariidae":        "鹤鸵科",
	"cathartidae":        "美洲鹫科",
	"certhiidae":         "旋木雀科",
	"cettiidae":          "树莺科",
	"chaetopidae":        "岩鸫科",
	"charadriidae":       "鸻科",
	"chionidae":          "鞘嘴鸥科",
	"chloropseidae":      "叶鹎科",
	"ciconiidae":         "鹳科",
	"cinclidae":          "河乌科",
	"cinclosomatidae":    "鹑鸫科",
	"cisticolidae":       "扇尾莺科",
	"climacteridae":      "澳洲旋木雀科",
	"cnemophilidae":      "极乐鸫科",
	"coliidae":           "鼠鸟科",
	"columbidae":         "鸠鸽科",
	"conopophagidae":     "食蚊鸟科",
	"coraciidae":         "佛法僧科",
	"corcoracidae":       "澳鸦科",
	"corvidae":           "鸦科",
	"cotingidae":         "伞鸟科",
	"cracidae":           "凤冠雉科",
	"cuculidae":          "杜鹃科",
	"dasyornithidae":     "须莺科",
	"dicaeidae":          "啄花鸟科",
	"dicruridae":         "卷尾科",
	"diomedeidae":        "信天翁科",
	"donacobiidae":       "黑顶苇鹪鹩科",
	"dulidae":            "棕榈䳭科",
	"dromadidae":         "蟹鸻科",
	"elachuridae":        "丽星鹩鹛科",
	"emberizidae":        "鹀科",
	"erythrocercidae":    "黄鹟科",
	"estrildidae":        "梅花雀科",
	"eulacestomatidae":   "耕嘴雀科",
	"eupetidae":          "长颈鸫科",
	"eurylaimidae":       "阔嘴鸟科",
	"eurypygidae":        "日鳽科",
	"falconidae":         "隼科",
	"falcunculidae":      "鵙雀科",
	"formicariidae":      "蚁鸫科",
	"fregatidae":         "军舰鸟科",
	"fringillidae":       "燕雀科",
	"furnariidae":        "灶鸟科",
	"galbulidae":         "嗡鴷科",
	"gaviidae":           "潜鸟科",
	"glareolidae":        "燕鸻科",
	"grallariidae":       "蚁八色鸫科",
	"gruidae":            "鹤科",
	"haematopodidae":     "蛎鹬科",
	"heliornithidae":     "日鷉科",
	"hemiprocnidae":      "凤头雨燕科",
	"hirundinidae":       "燕科",
	"hydrobatidae":       "海燕科",
	"hyliidae":           "蓬背莺科",
	"hyliotidae":         "非莺科",
	"hylocitreidae":      "林啸鹟科",
	"hypocoliidae":       "连雀科",
	"ibidorhynchidae":    "鹮嘴鹬科",
	"icteridae":          "拟鹂科",
	"ifritidae":          "蓝顶鸫科",
	"indicatoridae":      "响蜜鴷科",
	"irenidae":           "和平鸟科",
	"jacanidae":          "雉鸻科",
	"laniidae":           "伯劳科",
	"laridae":            "鸥科",
	"leiothrichidae":     "噪鹛科",
	"leptosomidae":       "鹃鴗科",
	"locustellidae":      "蝗莺科",
	"lybiidae":           "非洲须鴷科",
	"machaerirhynchidae": "船嘴鹟科",
	"macrosphenidae":     "非长嘴莺科",
	"malaconotidae":      "丛伯劳科",
	"maluridae":          "细尾鹩莺科",
	"megalaimidae":       "拟啄木鸟科",
	"megapodiidae":       "冢雉科",
	"melampittidae":      "黑八色鸫科",
	"melanocharitidae":   "啄果鸟科",
	"melanopareiidae":    "月胸雀科",
	"meliphagidae":       "吸蜜鸟科",
	"menuridae":          "琴鸟科",
	"meropidae":          "蜂虎科",
	"mesitornithidae":    "拟鹑科",
	"mimidae":            "嘲鸫科",
	"mitrospingidae":     "帽唐纳雀科",
	"modulatricidae":     "斑喉鸫科",
	"mohoidae":           "夏威夷吸蜜鸟科",
	"mohouidae":          "黄头澳鸫科",
	"momotidae":          "翠鴗科",
	"monarchidae":        "王鹟科",
	"motacillidae":       "鹡鸰科",
	"muscicapidae":       "鹟科",
	"musophagidae":       "蕉鹃科",
	"nectariniidae":      "太阳鸟科",
	"neosittidae":        "澳䴓科",
	"nicatoridae":        "黄喉鵙科",
	"notiomystidae":      "缝合吸蜜鸟科",
	"numididae":          "珠鸡科",
	"nyctibiidae":        "林鸱科",
	"oceanitidae":        "南海燕科",
	"odontophoridae":     "齿鹑科",
	"onychorhynchidae":   "皇霸鹟科",
	"opisthocomidae":     "麝雉科",
	"oreoicidae":         "冠钟鹊科",
	"oriolidae":          "黄鹂科",
	"orthonychidae":      "刺尾鸫科",
	"otididae":           "鸨科",
	"oxyruncidae":        "尖喙鸟科",
	"pachycephalidae":    "啸鹟科",
	"pandionidae":        "鹗科",
	"panuridae":          "文须雀科",
	"paradisaeidae":      "极乐鸟科",
	"paradoxornithidae":  "鸦雀科",
	"paramythiidae":      "冠啄果鸟科",
	"pardalotidae":       "斑食蜜鸟科",
	"paridae":            "山雀科",
	"parulidae":          "森莺科",
	"passerellidae":      "美洲鹀科",
	"passeridae":         "雀科",
	"pedionomidae":       "领鹑科",
	"pelecanidae":        "鹈鹕科",
	"pellorneidae":       "幽鹛科",
	"petroicidae":        "鸲鹟科",
	"peucedramidae":      "绿森莺科",
	"phaenicophilidae":   "棕榈唐纳雀科",
	"phaethontidae":      "鹲科",
	"phalacrocoracidae":  "鸬鹚科",
	"phasianidae":        "雉科",
	"philepittidae":      "裸眉鸫科",
	"phoenicopteridae":   "红鹳科",
	"phoeniculidae":      "林戴胜科",
	"phylloscopidae":     "柳莺科",
	"picathartidae":      "岩鹛科",
	"picidae":            "啄木鸟科",
	"pipridae":           "娇鹟科",
	"pittidae":           "八色鸫科",
	"pityriasidae":       "棘头伯劳科",
	"platylophidae":      "凤头鸦科",
	"platysteiridae":     "疣眼鹟科",
	"ploceidae":          "织布鸟科",
	"pluvianellidae":     "麦哲伦鸻科",
	"pluvianidae":        "埃及鸻科",
	"pnoepygidae":        "鳞胸鹪鹛科",
	"podargidae":         "蛙口夜鹰科",
	"podicipedidae":      "鸊鷉科",
	"polioptilidae":      "蚋莺科",
	"pomatostomidae":     "澳鹛科",
	"procellariidae":     "鹱科",
	"promeropidae":       "非洲蜜鸟科",
	"prunellidae":        "岩鹨科",
	"psittacidae":        "鹦鹉科",
	"psittaculidae":      "长尾鹦鹉科",
	"psophiidae":         "喇叭鸟科",
	"psophodidae":        "绿鸫科",
	"pteroclidae":        "沙鸡科",
	"ptiliogonatidae":    "丝鹟科",
	"ptilonorhynchidae":  "园丁鸟科",
	"pycnonotidae":       "鹎科",
	"rallidae":           "秧鸡科",
	"ramphastidae":       "鵎鵼科",
	"recurvirostridae":   "反嘴鹬科",
	"regulidae":          "戴菊科",
	"remizidae":          "攀雀科",
	"rhagologidae":       "斑果猎鸟科",
	"rheidae":            "美洲鸵科",
	"rhinocryptidae":     "窜鸟科",
	"rhipiduridae":       "扇尾鹟科",
	"rhodinocichlidae":   "玫瑰胸唐纳雀科",
	"rhynochetidae":      "鹭鹤科",
	"rostratulidae":      "彩鹬科",
	"sagittariidae":      "蛇鹫科",
	"salpornithidae":     "斑旋木雀科",
	"sapayoidae":         "阔嘴娇鹟科",
	"sarothruridae":      "绒秧鸡科",
	"scolopacidae":       "鹬科",
	"scopidae":           "锤头鹳科",
	"semnornithidae":     "鵎鵼须鴷科",
	"sittidae":           "䴓科",
	"spheniscidae":       "企鹅科",
	"steatornithidae":    "油鸱科",
	"stenostiridae":      "仙鹟科",
	"stercorariidae":     "贼鸥科",
	"strigidae":          "鸱鸮科",
	"strigopidae":        "鸮鹦鹉科",
	"struthionidae":      "鸵鸟科",
	"sturnidae":          "椋鸟科",
	"sulidae":            "鲣鸟科",
	"sylviidae":          "莺科",
	"teretistridae":      "古巴莺科",
	"thamnophilidae":     "蚁鸟科",
	"thinocoridae":       "籽鹬科",
	"thraupidae":         "唐纳雀科",
	"threskiornithidae":  "鹮科",
	"tichodromidae":      "旋壁雀科",
	"timaliidae":         "画眉科",
	"tinamidae":          "䳍科",
	"tityridae":          "蒂泰雀科",
	"todidae":            "短尾鴗科",
	"trochilidae":        "蜂鸟科",
	"troglodytidae":      "鹪鹩科",
	"trogonidae":         "咬鹃科",
	"turdidae":           "鸫科",
	"turnicidae":         "三趾鹑科",
	"tyrannidae":         "霸鹟科",
	"tytonidae":          "草鸮科",
	"upupidae":           "戴胜科",
	"urocynchramidae":    "朱鹀科",
	"vangidae":           "钩嘴鵙科",
	"viduidae":           "维达雀科",
	"vireonidae":         "绿鹃科",
	"zeledoniidae":       "冠鹩莺科",
	"zosteropidae":       "绣眼鸟科",
}

var familyComZhMap = map[string]string{
	"ostriches":                                  "鸵鸟科",
	"cassowaries and emu":                        "鹤鸵科",
	"rheas":                                      "美洲鸵科",
	"tinamous":                                   "䳍科",
	"kiwis":                                      "几维科",
	"ducks, geese, and waterfowl":                "鸭科",
	"pheasants, grouse, and allies":              "雉科",
	"flamingos":                                  "红鹳科",
	"grebes":                                     "鸊鷉科",
	"pigeons and doves":                          "鸠鸽科",
	"sandgrouse":                                 "沙鸡科",
	"bustards":                                   "鸨科",
	"cuckoos":                                    "杜鹃科",
	"nightjars and allies":                       "夜鹰科",
	"swifts":                                     "雨燕科",
	"rails, gallinules, and coots":               "秧鸡科",
	"cranes":                                     "鹤科",
	"thick-knees":                                "石鸻科",
	"oystercatchers":                             "蛎鹬科",
	"ibisbill":                                   "鹮嘴鹬科",
	"stilts and avocets":                         "反嘴鹬科",
	"plovers and lapwings":                       "鸻科",
	"painted-snipes":                             "彩鹬科",
	"jacanas":                                    "雉鸻科",
	"sandpipers and allies":                      "鹬科",
	"buttonquail":                                "三趾鹑科",
	"pratincoles and coursers":                   "燕鸻科",
	"skuas and jaegers":                          "贼鸥科",
	"auks, murres, and puffins":                  "海雀科",
	"gulls, terns, and skimmers":                 "鸥科",
	"tropicbirds":                                "鹲科",
	"loons":                                      "潜鸟科",
	"albatrosses":                                "信天翁科",
	"northern storm-petrels":                     "海燕科",
	"shearwaters and petrels":                    "鹱科",
	"storks":                                     "鹳科",
	"frigatebirds":                               "军舰鸟科",
	"boobies and gannets":                        "鲣鸟科",
	"anhingas":                                   "蛇鹈科",
	"cormorants and shags":                       "鸬鹚科",
	"pelicans":                                   "鹈鹕科",
	"herons, egrets, and bitterns":               "鹭科",
	"ibises and spoonbills":                      "鹮科",
	"osprey":                                     "鹗科",
	"hawks, eagles, and kites":                   "鹰科",
	"barn-owls":                                  "草鸮科",
	"owls":                                       "鸱鸮科",
	"trogons":                                    "咬鹃科",
	"hornbills":                                  "犀鸟科",
	"hoopoes":                                    "戴胜科",
	"kingfishers":                                "翠鸟科",
	"bee-eaters":                                 "蜂虎科",
	"rollers":                                    "佛法僧科",
	"asian barbets":                              "拟啄木鸟科",
	"woodpeckers":                                "啄木鸟科",
	"falcons and caracaras":                      "隼科",
	"old world parrots":                          "长尾鹦鹉科",
	"pittas":                                     "八色鸫科",
	"cuckooshrikes":                              "山椒鸟科",
	"whistlers and allies":                       "啸鹟科",
	"old world orioles":                          "黄鹂科",
	"woodswallows, bellmagpies, and allies":      "燕鵙科",
	"vireos, shrike-babblers, and erpornis":      "绿鹃科",
	"vangas, helmetshrikes, and allies":          "钩嘴鵙科",
	"drongos":                                    "卷尾科",
	"fantails":                                   "扇尾鹟科",
	"shrikes":                                    "伯劳科",
	"crows, jays, and magpies":                   "鸦科",
	"monarch flycatchers":                        "王鹟科",
	"fairy flycatchers":                          "仙鹟科",
	"tits, chickadees, and titmice":              "山雀科",
	"penduline-tits":                             "攀雀科",
	"larks":                                      "百灵科",
	"bearded reedling":                           "文须雀科",
	"cisticolas and allies":                      "扇尾莺科",
	"reed warblers and allies":                   "苇莺科",
	"grassbirds and allies":                      "蝗莺科",
	"swallows":                                   "燕科",
	"bulbuls":                                    "鹎科",
	"leaf warblers":                              "柳莺科",
	"bush warblers and allies":                   "树莺科",
	"long-tailed tits":                           "长尾山雀科",
	"sylviid warblers and allies":                "莺科",
	"sylviid warblers, parrotbills, and allies":  "莺科",
	"parrotbills":                                "鸦雀科",
	"white-eyes, yuhinas, and allies":            "绣眼鸟科",
	"tree-babblers, scimitar-babblers, and allies": "画眉科",
	"ground babblers and allies":                 "幽鹛科",
	"laughingthrushes and allies":                "噪鹛科",
	"kinglets":                                   "戴菊科",
	"wallcreeper":                                "旋壁雀科",
	"nuthatches":                                 "䴓科",
	"treecreepers":                               "旋木雀科",
	"wrens":                                      "鹪鹩科",
	"dippers":                                    "河乌科",
	"starlings":                                  "椋鸟科",
	"thrushes and allies":                        "鸫科",
	"old world flycatchers":                      "鹟科",
	"waxwings":                                   "太平鸟科",
	"flowerpeckers":                              "啄花鸟科",
	"sunbirds and spiderhunters":                 "太阳鸟科",
	"fairy-bluebirds":                            "和平鸟科",
	"leafbirds":                                  "叶鹎科",
	"accentors":                                  "岩鹨科",
	"weavers and allies":                         "织布鸟科",
	"waxbills and allies":                        "梅花雀科",
	"old world sparrows":                         "雀科",
	"wagtails and pipits":                        "鹡鸰科",
	"finches, euphonias, and allies":             "燕雀科",
	"longspurs and snow buntings":                "铁爪鹀科",
	"old world buntings":                         "鹀科",
	"penguins":                                   "企鹅科",
	"hummingbirds":                               "蜂鸟科",
	"tyrant flycatchers":                         "霸鹟科",
	"tanagers and allies":                        "唐纳雀科",
	"new world warblers":                         "森莺科",
	"new world sparrows":                         "美洲鹀科",
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
