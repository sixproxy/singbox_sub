package model

type Outbound struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

func (b Outbound) GetTag() string  { return b.Tag }
func (b Outbound) GetType() string { return b.Type }

type OutboundConfig interface {
	GetTag() string
	GetType() string
	Validate() error
}

// Direct出口
type DirectOutbound struct {
	Outbound
}

func (d DirectOutbound) Validate() error {
	return nil
}

type SocksOutbound struct {
	Outbound
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
}

func (s SocksOutbound) Validate() error {
	return nil
}
