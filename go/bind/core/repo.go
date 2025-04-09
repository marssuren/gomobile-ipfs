/*
文件概览：repo.go
这个文件定义了IPFS仓库(Repository)的Go绑定和管理。它提供了以下核心功能：
1. 仓库初始化、打开和关闭
2. 仓库配置的获取和设置
3. IPFS插件系统的加载与初始化
4. 线程安全的插件管理

IPFS插件系统允许扩展IPFS核心功能，如添加新的数据存储后端、新的网络传输协议等。
在移动环境中，这个加载机制被特别优化以确保线程安全和资源有效利用。
*/

package core

import (
	// 标准库导入
	"path/filepath" // 处理文件路径
	"sync"          // 提供同步原语，如互斥锁

	// 项目内部包
	ipfs_mobile "github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/ipfsmobile" // 移动平台IPFS实现

	// IPFS核心包
	ipfs_loader "github.com/ipfs/kubo/plugin/loader" // IPFS插件加载器
	ipfs_repo "github.com/ipfs/kubo/repo"            // IPFS仓库接口
	ipfs_fsrepo "github.com/ipfs/kubo/repo/fsrepo"   // 基于文件系统的IPFS仓库实现
)

var (
	// 全局变量，用于插件管理
	muPlugins sync.Mutex                // 保护plugins变量的互斥锁
	plugins   *ipfs_loader.PluginLoader // 全局插件加载器实例
)

// Repo 结构体包装了移动平台的IPFS仓库
type Repo struct {
	mr *ipfs_mobile.RepoMobile // 指向移动平台IPFS仓库的指针
}

// RepoIsInitialized 检查指定路径的IPFS仓库是否已初始化
func RepoIsInitialized(path string) bool {
	// 调用IPFS标准库检查仓库是否已初始化
	return ipfs_fsrepo.IsInitialized(path)
}

// InitRepo 在指定路径初始化IPFS仓库
func InitRepo(path string, cfg *Config) error {
	// 加载插件，确保初始化仓库前插件系统已就绪
	if _, err := loadPlugins(path); err != nil {
		return err
	}

	// 使用配置初始化仓库
	return ipfs_fsrepo.Init(path, cfg.getConfig())
}

// OpenRepo 打开现有的IPFS仓库
func OpenRepo(path string) (*Repo, error) {
	// 加载插件，确保打开仓库前插件系统已就绪
	if _, err := loadPlugins(path); err != nil {
		return nil, err
	}

	// 打开标准IPFS仓库
	irepo, err := ipfs_fsrepo.Open(path)
	if err != nil {
		return nil, err
	}

	// 创建移动平台适用的仓库包装
	mRepo := ipfs_mobile.NewRepoMobile(path, irepo)
	return &Repo{mRepo}, nil
}

// GetRootPath 返回仓库的根路径
func (r *Repo) GetRootPath() string {
	return r.mr.Path()
}

// SetConfig 设置仓库配置
func (r *Repo) SetConfig(c *Config) error {
	return r.mr.Repo.SetConfig(c.getConfig())
}

// GetConfig 获取仓库配置
func (r *Repo) GetConfig() (*Config, error) {
	// 获取底层仓库配置
	cfg, err := r.mr.Repo.Config()
	if err != nil {
		return nil, err
	}

	// 包装为高级配置对象
	return &Config{cfg}, nil
}

// Close 关闭仓库
func (r *Repo) Close() error {
	return r.mr.Close()
}

// getRepo 返回底层IPFS仓库接口
// 这是一个非导出方法(小写开头)，只能在包内使用
func (r *Repo) getRepo() ipfs_repo.Repo {
	return r.mr
}

// loadPlugins 加载IPFS插件系统
func loadPlugins(repoPath string) (*ipfs_loader.PluginLoader, error) {
	// 加锁确保多线程安全
	muPlugins.Lock()
	defer muPlugins.Unlock() // 确保函数退出时解锁

	// 如果插件已加载，直接返回现有实例（单例模式）
	if plugins != nil {
		return plugins, nil
	}

	// 构建插件目录路径
	// 默认IPFS插件存放在仓库的"plugins"子目录
	pluginpath := filepath.Join(repoPath, "plugins")

	// 创建新的插件加载器
	lp, err := ipfs_loader.NewPluginLoader(pluginpath)
	if err != nil {
		return nil, err
	}

	// 初始化插件系统
	// 这会查找和加载所有可用插件的元数据
	if err = lp.Initialize(); err != nil {
		return nil, err
	}

	// 注入插件
	// 这会将插件实际集成到IPFS系统中，使其功能可用
	if err = lp.Inject(); err != nil {
		return nil, err
	}

	// 保存全局实例并返回
	plugins = lp
	return lp, nil
}
