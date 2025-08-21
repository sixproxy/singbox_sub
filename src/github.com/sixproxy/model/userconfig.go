package model

// UserConfig 用户配置结构 (对应 config.yaml)
type UserConfig struct {
	Subs         []Sub         `yaml:"subs,omitempty"`
	Experimental *Experimental `yaml:"experimental,omitempty"`
	DNS          *UserDNS      `yaml:"dns,omitempty"`
	GitHub       *GitHubConfig `yaml:"github,omitempty"`
}

// UserDNS 用户DNS配置
type UserDNS struct {
	ClientSubnet string `yaml:"client_subnet,omitempty"`
	Strategy     string `yaml:"strategy,omitempty"`
	Final        string `yaml:"final,omitempty"`
	AutoOptimize bool   `yaml:"auto_optimize,omitempty"` // 是否自动优化client_subnet
}

// GitHubConfig GitHub配置
type GitHubConfig struct {
	MirrorURL       string   `yaml:"mirror_url,omitempty"`       // 主要镜像地址
	FallbackMirrors []string `yaml:"fallback_mirrors,omitempty"` // 备用镜像列表
}
