package model

import "fmt"

// URLTest类型的Outbound
type URLTestOutbound struct {
	Outbound
	Outbounds                 []string `json:"outbounds,omitempty"`
	Filters                   []Filter `json:"filter,omitempty"`
	URL                       string   `json:"url,omitempty"`
	Interval                  string   `json:"interval,omitempty"`
	Tolerance                 int      `json:"tolerance,omitempty"`
	InterruptExistConnections bool     `json:"interrupt_exist_connections,omitempty"`
}

func (u URLTestOutbound) Validate() error {
	if len(u.Outbounds) == 0 {
		return fmt.Errorf("urltest outbound must have outbounds")
	}
	return nil
}
