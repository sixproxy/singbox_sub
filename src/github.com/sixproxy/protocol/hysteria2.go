package protocol

import "singbox_sub/src/github.com/sixproxy/model"

var _ Parser = (*Hysteria2Parser)(nil)

type Hysteria2Parser struct{}

func init() {
	parsers["hysteria2"] = &Hysteria2Parser{}
}

func (h Hysteria2Parser) Proto() string {
	return "hysteria2"
}

func (h Hysteria2Parser) Parse(data string) (string, error) {
	// 每次 new 一个 session，存放本次解析的临时状态
	s := &Hy2ParseSession{}
	return s.parse(data)
}

type Hy2ParseSession struct {
	data   string
	Config model.Hysteria2Config
}

func (h *Hy2ParseSession) parse(data string) (string, error) {
	if err := h.Config.SetData(data); err != nil {
		return "", err
	}

	return h.Config.String(), nil
}
