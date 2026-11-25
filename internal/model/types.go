package model

import "time"

// Taxon 表示 eBird 鸟类分类名录中的单个物种条目。
type Taxon struct {
	SpeciesCode   string   `json:"speciesCode"`
	ComName       string   `json:"comName"`
	SciName       string   `json:"sciName"`
	Category      string   `json:"category"`
	TaxonOrder    float64  `json:"taxonOrder"`
	Order         string   `json:"order"`
	FamilyCode    string   `json:"familyCode"`
	FamilyComName string   `json:"familyComName"`
	FamilySciName string   `json:"familySciName"`
	BandingCodes  []string `json:"bandingCodes,omitempty"`
}

// Observation 表示 eBird API v2 返回的单条近期观测或珍稀观测记录。
type Observation struct {
	SpeciesCode      string  `json:"speciesCode"`
	ComName          string  `json:"comName"`
	SciName          string  `json:"sciName"`
	LocID            string  `json:"locId"`
	LocName          string  `json:"locName"`
	ObsDt            string  `json:"obsDt"`
	HowMany          int     `json:"howMany"`
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	ObsValid         bool    `json:"obsValid"`
	ObsReviewed      bool    `json:"obsReviewed"`
	LocationPrivate  bool    `json:"locationPrivate"`
	SubID            string  `json:"subId"`
	ObsID            string  `json:"obsId"`
	Subnational1Code string  `json:"subnational1Code"`
	Subnational1Name string  `json:"subnational1Name"`
	CountryCode      string  `json:"countryCode"`
	CountryName      string  `json:"countryName"`
	UserDisplayName  string  `json:"userDisplayName"`
	Evidence         string  `json:"evidence"`
	HasComments      bool    `json:"hasComments"`
	HasRichMedia     bool    `json:"hasRichMedia"`
}

// ChecklistObs 表示单份 eBird 观鸟清单中的单个鸟种观测项。
type ChecklistObs struct {
	SpeciesCode string         `json:"speciesCode"`
	ComName     string         `json:"comName,omitempty"`
	SciName     string         `json:"sciName,omitempty"`
	HowManyStr  string         `json:"howManyStr"`
	Comments    string         `json:"comments"`
	MediaCounts map[string]int `json:"mediaCounts"`
	ObsID       string         `json:"obsId"`
}

// ChecklistDetail 表示 eBird /v2/product/checklist/view/{subId} 的完整清单详情。
type ChecklistDetail struct {
	SubID             string         `json:"subId"`
	ProtocolID        string         `json:"protocolId"`
	LocID             string         `json:"locId"`
	DurationHrs       float64        `json:"durationHrs"`
	EffortDistanceKm  float64        `json:"effortDistanceKm"`
	AllObsReported    bool           `json:"allObsReported"`
	ObsDt             string         `json:"obsDt"`
	NumObservers      int            `json:"numObservers"`
	UserDisplayName   string         `json:"userDisplayName"`
	NumSpecies        int            `json:"numSpecies"`
	Subnational1Code  string         `json:"subnational1Code"`
	Obs               []ChecklistObs `json:"obs"`
}

// Hotspot 表示 eBird 观鸟热点。
type Hotspot struct {
	LocID             string  `json:"locId"`
	LocName           string  `json:"locName"`
	CountryCode       string  `json:"countryCode"`
	Subnational1Code  string  `json:"subnational1Code"`
	Lat               float64 `json:"lat"`
	Lng               float64 `json:"lng"`
	LatestObsDt       string  `json:"latestObsDt"`
	NumSpeciesAllTime int     `json:"numSpeciesAllTime"`
}

// RegionStat 表示某个地区单日 eBird 观测统计。
type RegionStat struct {
	NumChecklists   int `json:"numChecklists"`
	NumContributors int `json:"numContributors"`
	NumSpecies      int `json:"numSpecies"`
}

// RegionOption 表示预设地区选项（名称、eBird 区域码及中心坐标）。
type RegionOption struct {
	Code string
	Name string
	Lat  float64
	Lng  float64
}

// MediaPhoto 表示按需加载的在线图片资源（用于图鉴或现场实拍）。
type MediaPhoto struct {
	URL         string `json:"url"`
	ThumbURL    string `json:"thumbUrl"`
	Attribution string `json:"attribution"`
	License     string `json:"license"`
	Source      string `json:"source"`
}

// MediaSound 表示按需加载的在线鸟鸣音频流。
type MediaSound struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Quality   string `json:"quality"`
	Recordist string `json:"recordist"`
	Place     string `json:"place"`
	License   string `json:"license"`
	Source    string `json:"source"`
}

// WikiSummary 表示中文维基百科物种词条摘要。
type WikiSummary struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Extract     string `json:"extract"`
	PageURL     string `json:"pageUrl"`
	Thumbnail   string `json:"thumbnail"`
	Source      string `json:"source"`
}

// SpeciesMedia 表示「鸟类图鉴」使用的物种标准资料（按拉丁学名聚合）。
type SpeciesMedia struct {
	SciName               string       `json:"sciName"`
	ComName               string       `json:"comName"`
	INatTaxonID           int          `json:"inatTaxonId"`
	INatURL               string       `json:"inatUrl"`
	INatObservationsCount int          `json:"inatObservationsCount"`
	Photos                []MediaPhoto `json:"photos"`
	Sounds                []MediaSound `json:"sounds"`
	Wiki                  *WikiSummary `json:"wiki,omitempty"`
}

// FieldPhotoObs 表示区域内某次真实目击观测自带的现场实拍图与现场录音（严禁与图鉴混用）。
type FieldPhotoObs struct {
	ID           int64        `json:"id"`
	ComName      string       `json:"comName"`
	SciName      string       `json:"sciName"`
	ObservedOn   string       `json:"observedOn"`
	PlaceGuess   string       `json:"placeGuess"`
	Observer     string       `json:"observer"`
	QualityGrade string       `json:"qualityGrade"`
	ObsURL       string       `json:"obsUrl"`
	Lat          float64      `json:"lat"`
	Lng          float64      `json:"lng"`
	Photos       []MediaPhoto `json:"photos"`
	Sounds       []MediaSound `json:"sounds"`
}

// JournalEntry 表示用户个人的一条观鸟日志记录。
type JournalEntry struct {
	ID          string    `json:"id"`
	SpeciesCode string    `json:"speciesCode"`
	ComName     string    `json:"comName"`
	SciName     string    `json:"sciName"`
	Order       string    `json:"order"`
	FamilyName  string    `json:"familyName"`
	ObservedAt  string    `json:"observedAt"`
	Location    string    `json:"location"`
	Lat         float64   `json:"lat,omitempty"`
	Lng         float64   `json:"lng,omitempty"`
	Count       int       `json:"count"`
	Notes       string    `json:"notes"`
	PhotoPaths  []string  `json:"photoPaths,omitempty"`
	AudioPath   string    `json:"audioPath,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UnlockedBird 表示用户已解锁的个人鸟种记录（Life List）。
type UnlockedBird struct {
	SpeciesCode    string `json:"speciesCode"`
	ComName        string `json:"comName"`
	SciName        string `json:"sciName"`
	Order          string `json:"order"`
	FamilyName     string `json:"familyName"`
	FirstSeenAt    string `json:"firstSeenAt"`
	FirstLocation  string `json:"firstLocation"`
	FirstEntryID   string `json:"firstEntryId"`
	CoverPhotoPath string `json:"coverPhotoPath,omitempty"`
	TotalSightings int    `json:"totalSightings"`
}

// Settings 表示用户应用配置（API Key 优先保存在系统凭据管理器）。
type Settings struct {
	DefaultRegion   string `json:"defaultRegion"`
	Locale          string `json:"locale"`
	ThemeMode       string `json:"themeMode"` // "system", "light", "dark"
	EbirdKeyBackup  string `json:"ebirdKeyBackup,omitempty"`
	XenoKeyBackup   string `json:"xenoKeyBackup,omitempty"`
}
