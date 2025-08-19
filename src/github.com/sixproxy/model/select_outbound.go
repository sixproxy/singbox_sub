package model

import "fmt"

// Select类型的Outbound
type SelectorOutbound struct {
	Outbound
	Outbounds                 []string `json:"outbounds,omitempty"`
	Filters                   []Filter `json:"filter,omitempty"`
	Default                   string   `json:"default,omitempty"`
	InterruptExistConnections bool     `json:"interrupt_exist_connections,omitempty"`
}

func (s SelectorOutbound) Validate() error {
	if len(s.Outbounds) == 0 {
		return fmt.Errorf("select outbound must have outbounds")
	}
	return nil
}
