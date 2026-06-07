package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HotReloadConfig 热重载支持扩展
// 配置文件变更后自动重新加载，不影响正在运行的业务

// WatchConfig 启动配置文件监听（热重载）
// 通过轮询文件修改时间检测变更，不依赖 fsnotify 以减少依赖
func (cl *ConfigLoader) WatchConfig(stopCh <-chan struct{}) {
	if !cl.opts.HotReload {
		return
	}

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	lastModTimes := make(map[string]time.Time)
	files := []string{
		"realms.json",
		"techniques.json",
		"breakthrough.json",
	}

	// 初始化文件修改时间
	for _, f := range files {
		path := filepath.Join(cl.opts.DataDir, f)
		info, err := os.Stat(path)
		if err == nil {
			lastModTimes[path] = info.ModTime()
		}
	}

	cl.logger.Info("配置文件热重载监控已启动", "interval_seconds", 30)

	for {
		select {
		case <-stopCh:
			cl.logger.Info("配置文件监控已停止")
			return
		case <-ticker.C:
			changed := false
			for _, f := range files {
				path := filepath.Join(cl.opts.DataDir, f)
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				if lastModTime, ok := lastModTimes[path]; ok {
					if info.ModTime().After(lastModTime) {
						cl.logger.Info("检测到配置文件变更", "file", f)
						changed = true
						lastModTimes[path] = info.ModTime()
					}
				} else {
					lastModTimes[path] = info.ModTime()
				}
			}

			if changed {
				if err := cl.Reload(); err != nil {
					cl.logger.Error("热重载失败", "error", err)
				} else {
					cl.logger.Info("配置热重载成功")
				}
			}
		}
	}
}

// Reload 重新加载所有配置
func (cl *ConfigLoader) Reload() error {
	newLoader := NewConfigLoader(cl.logger, cl.filePath, cl.opts)
	if err := newLoader.Load(); err != nil {
		return fmt.Errorf("热重载失败: %w", err)
	}

	// 原子替换配置指针
	cl.config.mu.Lock()
	cl.config.Realms = newLoader.config.Realms
	cl.config.Techniques = newLoader.config.Techniques
	cl.config.Breakthrough = newLoader.config.Breakthrough
	cl.config.mu.Unlock()

	return nil
}

// DefaultConfigDir 返回默认配置目录
func DefaultConfigDir() string {
	// 优先尝试环境变量
	if dir := os.Getenv("CULTIVATION_DATA_DIR"); dir != "" {
		return dir
	}

	// 尝试从可执行文件路径推断
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "..", "internal", "data")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	// 默认值
	return "internal/data"
}
