package model

// LocationInfo IP地理位置信息
type LocationInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	Region  string `json:"region"` // 省份
	City    string `json:"city"`   // 城市
	ISP     string `json:"isp"`    // 运营商
}
