package model

import (
	"singbox_sub/src/github.com/sixproxy/service"
	"singbox_sub/src/github.com/sixproxy/util"
)

// CityMappingAdapter 实现service包中的CityMappingProvider接口
type CityMappingAdapter struct{}

// GetDefaultClientSubnet 实现接口
func (c *CityMappingAdapter) GetDefaultClientSubnet() string {
	return GetDefaultClientSubnet()
}

// GetCityISPSubnet 实现接口
func (c *CityMappingAdapter) GetCityISPSubnet(city, isp string) string {
	return GetCityISPSubnet(city, isp)
}

// GetFallbackSubnet 实现接口
func (c *CityMappingAdapter) GetFallbackSubnet(location *service.LocationInfo) string {
	// 转换类型
	utilLocation := &util.LocationInfo{
		IP:      location.IP,
		Country: location.Country,
		Region:  location.Region,
		City:    location.City,
		ISP:     location.ISP,
	}
	return GetFallbackSubnet(utilLocation)
}

// InferCityFromRegion 实现接口
func (c *CityMappingAdapter) InferCityFromRegion(region string) string {
	return InferCityFromRegion(region)
}

// GetCityNameCH 实现接口
func (c *CityMappingAdapter) GetCityNameCH(cityName string) string {
	return GetCityNameCH(cityName)
}

// GetDefaultCityByISP 实现接口
func (c *CityMappingAdapter) GetDefaultCityByISP(isp string) string {
	return GetDefaultCityByISP(isp)
}

// NormalizeISPName 实现接口
func (c *CityMappingAdapter) NormalizeISPName(isp string) string {
	return NormalizeISPName(isp)
}