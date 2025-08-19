package protocol

import "singbox_sub/src/github.com/sixproxy/model"

var _ Parser = (*trojanParser)(nil)

func init() {
	parsers["trojan"] = &trojanParser{}
}

type trojanParser struct {
}

func (p *trojanParser) Proto() string { return "trojan" }

func (p *trojanParser) Parse(raw string) (string, error) {
	s := &trojanParseSession{}
	return s.parse(raw)
}

type trojanParseSession struct {
	data   string
	Config model.TrojanConfig
}

func (p *trojanParseSession) parse(raw string) (string, error) {
	err := p.Config.SetData(raw)
	if err != nil {
		return "", err
	}
	return p.Config.String(), nil
}