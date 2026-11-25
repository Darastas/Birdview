package api

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/model"
)

// ChinaRegions 预置中国 34 个省级行政区及代表性坐标（用于区域观测与热点探索）。
var ChinaRegions = []model.RegionOption{
	{Code: "CN-11", Name: "北京 (CN-11)", Lat: 39.9042, Lng: 116.4074},
	{Code: "CN-31", Name: "上海 (CN-31)", Lat: 31.2304, Lng: 121.4737},
	{Code: "CN-44", Name: "广东 (CN-44)", Lat: 23.1291, Lng: 113.2644},
	{Code: "CN-53", Name: "云南 (CN-53)", Lat: 25.0406, Lng: 102.7123},
	{Code: "CN-33", Name: "浙江 (CN-33)", Lat: 30.2741, Lng: 120.1551},
	{Code: "CN-51", Name: "四川 (CN-51)", Lat: 30.5728, Lng: 104.0668},
	{Code: "CN-32", Name: "江苏 (CN-32)", Lat: 32.0603, Lng: 118.7969},
	{Code: "CN-35", Name: "福建 (CN-35)", Lat: 26.0745, Lng: 119.2965},
	{Code: "CN-12", Name: "天津 (CN-12)", Lat: 39.3434, Lng: 117.3616},
	{Code: "CN-13", Name: "河北 (CN-13)", Lat: 38.0428, Lng: 114.5149},
	{Code: "CN-14", Name: "山西 (CN-14)", Lat: 37.8706, Lng: 112.5489},
	{Code: "CN-15", Name: "内蒙古 (CN-15)", Lat: 40.8426, Lng: 111.7492},
	{Code: "CN-21", Name: "辽宁 (CN-21)", Lat: 41.8057, Lng: 123.4315},
	{Code: "CN-22", Name: "吉林 (CN-22)", Lat: 43.8171, Lng: 125.3235},
	{Code: "CN-23", Name: "黑龙江 (CN-23)", Lat: 45.8038, Lng: 126.5349},
	{Code: "CN-34", Name: "安徽 (CN-34)", Lat: 31.8206, Lng: 117.2272},
	{Code: "CN-36", Name: "江西 (CN-36)", Lat: 28.6829, Lng: 115.8582},
	{Code: "CN-37", Name: "山东 (CN-37)", Lat: 36.6512, Lng: 117.1201},
	{Code: "CN-41", Name: "河南 (CN-41)", Lat: 34.7466, Lng: 113.6254},
	{Code: "CN-42", Name: "湖北 (CN-42)", Lat: 30.5928, Lng: 114.3055},
	{Code: "CN-43", Name: "湖南 (CN-43)", Lat: 28.2282, Lng: 112.9388},
	{Code: "CN-45", Name: "广西 (CN-45)", Lat: 22.8170, Lng: 108.3665},
	{Code: "CN-46", Name: "海南 (CN-46)", Lat: 20.0440, Lng: 110.1983},
	{Code: "CN-50", Name: "重庆 (CN-50)", Lat: 29.5630, Lng: 106.5516},
	{Code: "CN-52", Name: "贵州 (CN-52)", Lat: 26.6477, Lng: 106.6302},
	{Code: "CN-54", Name: "西藏 (CN-54)", Lat: 29.6525, Lng: 91.1721},
	{Code: "CN-61", Name: "陕西 (CN-61)", Lat: 34.3416, Lng: 108.9398},
	{Code: "CN-62", Name: "甘肃 (CN-62)", Lat: 36.0611, Lng: 103.8343},
	{Code: "CN-63", Name: "青海 (CN-63)", Lat: 36.6171, Lng: 101.7782},
	{Code: "CN-64", Name: "宁夏 (CN-64)", Lat: 38.4872, Lng: 106.2309},
	{Code: "CN-65", Name: "新疆 (CN-65)", Lat: 43.8256, Lng: 87.6168},
	{Code: "HK", Name: "香港 (HK)", Lat: 22.3193, Lng: 114.1694},
	{Code: "MO", Name: "澳门 (MO)", Lat: 22.1987, Lng: 113.5439},
	{Code: "TW", Name: "台湾 (TW)", Lat: 25.0330, Lng: 121.5654},
	{Code: "CN", Name: "中国全国 (CN)", Lat: 35.8617, Lng: 104.1954},
}

// LookupRegionOption 根据区域码查找预设省市信息。
func LookupRegionOption(code string) (model.RegionOption, bool) {
	for _, r := range ChinaRegions {
		if strings.EqualFold(r.Code, code) {
			return r, true
		}
	}
	return model.RegionOption{}, false
}

// FetchTaxonomy 获取 eBird 全量物种级分类名录（免 API Key，默认 zh_SIM 大陆简体中文）。
func (c *Client) FetchTaxonomy(ctx context.Context, locale string) ([]model.Taxon, error) {
	if locale == "" {
		locale = "zh_SIM"
	}
	u := fmt.Sprintf("%s/ref/taxonomy/ebird?fmt=json&cat=species&locale=%s",
		c.EBirdBaseURL, url.QueryEscape(locale))
	var out []model.Taxon
	if err := c.getJSON(ctx, u, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchSpeciesList 获取指定国家或地区有记录的物种代码列表（需 eBird API Key）。
func (c *Client) FetchSpeciesList(ctx context.Context, apiKey, regionCode string) ([]string, error) {
	if apiKey == "" {
		return nil, errors.New("未配置 eBird API Key")
	}
	u := fmt.Sprintf("%s/product/spplist/%s", c.EBirdBaseURL, url.PathEscape(regionCode))
	var out []string
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": apiKey}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchRecentObservations 获取区域（或热点 locId）近期观测记录。
func (c *Client) FetchRecentObservations(ctx context.Context, apiKey, regionCode string, backDays, maxResults int, notable bool) ([]model.Observation, error) {
	if apiKey == "" {
		return nil, errors.New("查询区域观测记录需要配置 eBird API Key")
	}
	if backDays <= 0 || backDays > 30 {
		backDays = 14
	}
	if maxResults <= 0 {
		maxResults = 80
	}
	endpoint := "recent"
	detail := "simple"
	if notable {
		endpoint = "recent/notable"
		detail = "full"
	}
	u := fmt.Sprintf("%s/data/obs/%s/%s?sppLocale=zh_SIM&back=%d&maxResults=%d&detail=%s",
		c.EBirdBaseURL, url.PathEscape(regionCode), endpoint, backDays, maxResults, detail)
	var out []model.Observation
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": apiKey}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchNearbyObservations 根据经纬度与半径（km，最大50）获取周边近期观鸟记录。
func (c *Client) FetchNearbyObservations(ctx context.Context, apiKey string, lat, lng float64, distKm, backDays, maxResults int) ([]model.Observation, error) {
	if apiKey == "" {
		return nil, errors.New("查询附近观测记录需要配置 eBird API Key")
	}
	if distKm <= 0 || distKm > 50 {
		distKm = 25
	}
	if backDays <= 0 || backDays > 30 {
		backDays = 14
	}
	if maxResults <= 0 {
		maxResults = 80
	}
	u := fmt.Sprintf("%s/data/obs/geo/recent?lat=%.5f&lng=%.5f&dist=%d&back=%d&maxResults=%d&sppLocale=zh_SIM",
		c.EBirdBaseURL, lat, lng, distKm, backDays, maxResults)
	var out []model.Observation
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": apiKey}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchChecklistDetail 获取指定提交清单 (subId) 的完整现场明细与媒体计数。
func (c *Client) FetchChecklistDetail(ctx context.Context, apiKey, subID string) (*model.ChecklistDetail, error) {
	if apiKey == "" {
		return nil, errors.New("查询清单详情需要配置 eBird API Key")
	}
	u := fmt.Sprintf("%s/product/checklist/view/%s", c.EBirdBaseURL, url.PathEscape(subID))
	var out model.ChecklistDetail
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": apiKey}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FetchHotspots 获取某地区最近活跃的观鸟热点。
func (c *Client) FetchHotspots(ctx context.Context, apiKey, regionCode string, backDays int) ([]model.Hotspot, error) {
	if apiKey == "" {
		return nil, errors.New("查询观鸟热点需要配置 eBird API Key")
	}
	if backDays <= 0 || backDays > 30 {
		backDays = 30
	}
	u := fmt.Sprintf("%s/ref/hotspot/%s?fmt=json&back=%d",
		c.EBirdBaseURL, url.PathEscape(regionCode), backDays)
	var out []model.Hotspot
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": apiKey}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchRegionStats 获取某地区指定日期（默认昨日）的观鸟活动统计。
func (c *Client) FetchRegionStats(ctx context.Context, apiKey, regionCode string, day time.Time) (*model.RegionStat, error) {
	if apiKey == "" {
		return nil, errors.New("查询区域统计需要配置 eBird API Key")
	}
	if day.IsZero() {
		day = time.Now().AddDate(0, 0, -1)
	}
	u := fmt.Sprintf("%s/product/stats/%s/%d/%d/%d",
		c.EBirdBaseURL, url.PathEscape(regionCode), day.Year(), int(day.Month()), day.Day())
	var out model.RegionStat
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": apiKey}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyEbirdKey 验证 eBird API Key 有效性并测量响应耗时。
func (c *Client) VerifyEbirdKey(ctx context.Context, apiKey string) (time.Duration, error) {
	if strings.TrimSpace(apiKey) == "" {
		return 0, errors.New("API Key 不能为空")
	}
	start := time.Now()
	u := fmt.Sprintf("%s/ref/region/list/subnational1/CN", c.EBirdBaseURL)
	var regions []struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := c.getJSON(ctx, u, map[string]string{"X-eBirdApiToken": strings.TrimSpace(apiKey)}, &regions); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
