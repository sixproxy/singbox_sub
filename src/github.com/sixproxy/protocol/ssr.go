package protocol

import "singbox_sub/src/github.com/sixproxy/model"

// 让编译器检查：*ssrParser 实现了 Parser 接口
var _ Parser = (*ssrParser)(nil)

// 单例，包加载时自动注册
func init() {
	parsers["ssr"] = &ssrParser{}
}

type ssrParser struct {
}

func (p *ssrParser) Proto() string { return "ssr" }

func (p *ssrParser) Parse(raw string) (string, error) {

	// 每次 new 一个 session，存放本次解析的临时状态
	s := &ssrParseSession{}
	return s.parse(raw)

}

type ssrParseSession struct {
	data   string
	Config model.SsrConfig
}

func (p *ssrParseSession) parse(raw string) (string, error) {
	err := p.Config.SetData(raw)
	if err != nil {
		return "", err
	}
	return p.Config.String(), nil
}
