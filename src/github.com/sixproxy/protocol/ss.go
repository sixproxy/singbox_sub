package protocol

import "singbox_sub/src/github.com/sixproxy/model"

// 让编译器检查：*ssParser 实现了 Parser 接口
var _ Parser = (*ssParser)(nil)

// 单例，包加载时自动注册
func init() {
	parsers["ss"] = &ssParser{}
}

type ssParser struct {
}

func (p *ssParser) Proto() string { return "ss" }

func (p *ssParser) Parse(raw string) (string, error) {

	// 每次 new 一个 session，存放本次解析的临时状态
	s := &ssParseSession{}
	return s.parse(raw)

}

type ssParseSession struct {
	data   string
	Config model.SsConfig
}

func (p *ssParseSession) parse(raw string) (string, error) {
	err := p.Config.SetData(raw)
	if err != nil {
		return "", err
	}
	return p.Config.String(), nil
}
