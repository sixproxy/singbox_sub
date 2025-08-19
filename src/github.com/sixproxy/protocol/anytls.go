package protocol

import "singbox_sub/src/github.com/sixproxy/model"

var _ Parser = (*AnytlsParser)(nil)

type AnytlsParser struct{}

func init() {
	parsers["anytls"] = &AnytlsParser{}
}

func (a AnytlsParser) Proto() string {
	return "anytls"
}

func (a AnytlsParser) Parse(data string) (string, error) {
	// Create a new session for parsing
	s := &AnytlsParseSession{}
	return s.parse(data)
}

type AnytlsParseSession struct {
	data   string
	Config model.AnytlsConfig
}

func (a *AnytlsParseSession) parse(data string) (string, error) {
	if err := a.Config.SetData(data); err != nil {
		return "", err
	}

	if err := a.Config.Validate(); err != nil {
		return "", err
	}

	return a.Config.String(), nil
}