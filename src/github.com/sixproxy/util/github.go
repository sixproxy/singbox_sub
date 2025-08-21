package util

import (
	"fmt"
	"net/http"
	"singbox_sub/src/github.com/sixproxy/logger"
	"strings"
	"time"
)

// GitHubMirrorManager GitHub镜像管理器
type GitHubMirrorManager struct {
	primaryMirror   string
	fallbackMirrors []string
	client          *http.Client
}

// NewGitHubMirrorManager 创建GitHub镜像管理器  
func NewGitHubMirrorManager(mirrorURL string, fallbackMirrors []string) *GitHubMirrorManager {
	return &GitHubMirrorManager{
		primaryMirror:   mirrorURL,
		fallbackMirrors: fallbackMirrors,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAPIURL 获取GitHub API URL，支持镜像
func (gm *GitHubMirrorManager) GetAPIURL(repo string, endpoint string) string {
	baseURL := "https://api.github.com"
	
	// 如果配置了主要镜像，优先使用
	if gm.primaryMirror != "" {
		if strings.HasSuffix(gm.primaryMirror, "/") {
			return gm.primaryMirror + "https://api.github.com/repos/" + repo + endpoint
		} else {
			return gm.primaryMirror + "/https://api.github.com/repos/" + repo + endpoint
		}
	}
	
	// 否则使用原始GitHub API
	return baseURL + "/repos/" + repo + endpoint
}

// GetDownloadURL 转换下载URL为镜像URL
func (gm *GitHubMirrorManager) GetDownloadURL(originalURL string) string {
	// 如果配置了主要镜像，转换下载URL
	if gm.primaryMirror != "" {
		return gm.convertToMirrorURL(originalURL, gm.primaryMirror)
	}
	
	// 否则返回原始URL
	return originalURL
}

// convertToMirrorURL 转换URL为镜像URL
func (gm *GitHubMirrorManager) convertToMirrorURL(originalURL, mirrorBase string) string {
	if strings.HasSuffix(mirrorBase, "/") {
		return mirrorBase + originalURL
	} else {
		return mirrorBase + "/" + originalURL
	}
}

// TestMirror 测试镜像可用性
func (gm *GitHubMirrorManager) TestMirror(mirrorURL string) error {
	// 测试访问GitHub主页
	testURL := gm.convertToMirrorURL("https://github.com", mirrorURL)
	
	req, err := http.NewRequest("HEAD", testURL, nil)
	if err != nil {
		return err
	}
	
	resp, err := gm.client.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接到镜像 %s: %v", mirrorURL, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("镜像 %s 返回错误状态码: %d", mirrorURL, resp.StatusCode)
	}
	
	return nil
}

// FindAvailableMirror 查找可用的镜像
func (gm *GitHubMirrorManager) FindAvailableMirror() (string, error) {
	// 如果配置了主要镜像，先测试它
	if gm.primaryMirror != "" {
		if err := gm.TestMirror(gm.primaryMirror); err == nil {
			logger.Info("使用配置的GitHub镜像: %s", gm.primaryMirror)
			return gm.primaryMirror, nil
		} else {
			logger.Warn("配置的GitHub镜像不可用: %v", err)
		}
	}
	
	// 测试备用镜像
	for _, mirror := range gm.fallbackMirrors {
		if err := gm.TestMirror(mirror); err == nil {
			logger.Info("找到可用的GitHub镜像: %s", mirror)
			return mirror, nil
		} else {
			logger.Debug("镜像 %s 不可用: %v", mirror, err)
		}
	}
	
	// 如果所有镜像都不可用，使用原始GitHub
	logger.Warn("所有GitHub镜像都不可用，使用原始GitHub API")
	return "", nil
}

// FetchWithMirror 使用镜像获取数据
func (gm *GitHubMirrorManager) FetchWithMirror(url string) (*http.Response, error) {
	var lastErr error
	
	// 尝试使用主要镜像
	if gm.primaryMirror != "" {
		mirrorURL := gm.GetAPIURL("", "")
		if strings.Contains(url, "api.github.com") {
			// 替换API URL
			mirrorURL = strings.Replace(url, "https://api.github.com", strings.TrimSuffix(gm.primaryMirror, "/")+"https://api.github.com", 1)
		} else {
			mirrorURL = gm.GetDownloadURL(url)
		}
		
		resp, err := gm.client.Get(mirrorURL)
		if err == nil && resp.StatusCode < 400 {
			logger.Debug("使用镜像成功获取: %s", mirrorURL)
			return resp, nil
		} else {
			if resp != nil {
				resp.Body.Close()
			}
			lastErr = err
			logger.Debug("镜像访问失败: %v", err)
		}
	}
	
	// 尝试备用镜像
	for _, mirror := range gm.fallbackMirrors {
		var mirrorURL string
		if strings.Contains(url, "api.github.com") {
			mirrorURL = strings.Replace(url, "https://api.github.com", strings.TrimSuffix(mirror, "/")+"https://api.github.com", 1)
		} else {
			mirrorURL = gm.convertToMirrorURL(url, mirror)
		}
		
		resp, err := gm.client.Get(mirrorURL)
		if err == nil && resp.StatusCode < 400 {
			logger.Debug("使用备用镜像成功获取: %s", mirrorURL)
			return resp, nil
		} else {
			if resp != nil {
				resp.Body.Close()
			}
			lastErr = err
			logger.Debug("备用镜像 %s 访问失败: %v", mirror, err)
		}
	}
	
	// 最后尝试原始URL
	logger.Debug("尝试使用原始GitHub URL: %s", url)
	resp, err := gm.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("所有镜像都失败，原始URL也失败: %v (最后一个镜像错误: %v)", err, lastErr)
	}
	
	return resp, nil
}


// testMirrorConnectivity 测试镜像连通性
func testMirrorConnectivity(mirrorURL string) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	// 构造测试URL - 使用具体的GitHub文件URL来测试
	var testURL string
	if strings.HasSuffix(mirrorURL, "/") {
		testURL = mirrorURL + "https://raw.githubusercontent.com/sixproxy/singbox_sub/main/README.md"
	} else {
		testURL = mirrorURL + "/https://raw.githubusercontent.com/sixproxy/singbox_sub/main/README.md"
	}
	
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		logger.Debug("创建请求失败 %s: %v", mirrorURL, err)
		return false
	}
	
	// 设置User-Agent以避免被拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("连接失败 %s: %v", mirrorURL, err)
		return false
	}
	defer resp.Body.Close()
	
	// GitHub镜像检测逻辑：
	// 200: 完全正常，内容已获取
	// 304: 内容未修改（缓存有效），也是可用的
	// 其他状态码: 不可用
	isAvailable := resp.StatusCode == 200 || resp.StatusCode == 304
	logger.Debug("镜像 %s 测试结果: %d (%t)", mirrorURL, resp.StatusCode, isAvailable)
	return isAvailable
}