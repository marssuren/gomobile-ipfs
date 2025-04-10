/*
文件概览：go/pkg/ipfsmobile/node.go
这个文件定义了移动平台IPFS节点的实现。它是gomobile-ipfs的核心部分，负责：
1. 定义IPFS节点的配置结构和选项
2. 提供移动平台优化的IPFS节点实现
3. 实现HTTP API和网关服务
4. 封装和简化IPFS核心功能供移动应用使用

该文件是连接IPFS核心库(kubo)和移动平台的桥梁，提供了适合移动环境的API接口和性能优化。
对蓝牙、pubsub等移动优化功能的支持就是通过这里的配置和实现来启用的。
*/

// node包提供移动平台IPFS节点的核心实现
package node

import (
	// 导入必要的标准库
	"context"       // 用于上下文管理
	"fmt"           // 用于格式化错误消息
	"net"           // 提供网络连接接口
	"os"            // 用于文件操作
	"path/filepath" // 用于路径处理
	"runtime/debug" // 用于获取堆栈信息
	"time"          // 用于时间戳

	// 导入IPFS核心组件
	ipfs_oldcmds "github.com/ipfs/kubo/commands"       // IPFS命令接口
	ipfs_core "github.com/ipfs/kubo/core"              // IPFS核心实现
	ipfs_corehttp "github.com/ipfs/kubo/core/corehttp" // IPFS HTTP接口
	ipfs_p2p "github.com/ipfs/kubo/core/node/libp2p"   // IPFS网络层配置
	p2p_host "github.com/libp2p/go-libp2p/core/host"   // libp2p主机接口

	// 导入日志包
	logging "github.com/ipfs/go-log" // IPFS日志系统
)

// 创建日志记录器
var log = logging.Logger("ipfs-mobile")

// IpfsConfig定义IPFS节点的配置选项
type IpfsConfig struct {
	// 网络主机配置，控制P2P通信层
	HostConfig *HostConfig
	// 主机选项，定义如何构建libp2p主机
	HostOption ipfs_p2p.HostOption

	// 路由配置，控制内容和节点发现
	RoutingConfig *RoutingConfig
	// 路由选项，定义如何构建DHT等路由系统
	RoutingOption ipfs_p2p.RoutingOption

	// 移动平台仓库实现，存储IPFS数据
	RepoMobile *RepoMobile
	// 额外选项映射，用于启用/禁用特定功能
	ExtraOpts map[string]bool
}

// fillDefault为配置填充默认值
// 确保配置对象包含所有必需的字段
func (c *IpfsConfig) fillDefault() error {
	// 仓库是必需的，不能为空
	if c.RepoMobile == nil {
		return fmt.Errorf("repo cannot be nil")
	}

	// 如果额外选项为空，创建空映射
	if c.ExtraOpts == nil {
		c.ExtraOpts = make(map[string]bool)
	}

	// 默认使用DHT(分布式哈希表)作为路由选项
	if c.RoutingOption == nil {
		c.RoutingOption = ipfs_p2p.DHTOption
	}

	// 如果没有路由配置，创建默认配置
	if c.RoutingConfig == nil {
		c.RoutingConfig = &RoutingConfig{}
	}

	// 默认使用标准主机选项
	if c.HostOption == nil {
		c.HostOption = ipfs_p2p.DefaultHostOption
	}

	// 如果没有主机配置，创建默认配置
	if c.HostConfig == nil {
		c.HostConfig = &HostConfig{}
	}

	return nil
}

// IpfsMobile是移动平台IPFS节点实现
// 封装了标准IPFS节点并添加移动优化功能
type IpfsMobile struct {
	// 嵌入IPFS核心节点
	*ipfs_core.IpfsNode
	// 引用移动平台仓库
	Repo *RepoMobile

	// 命令上下文，用于HTTP API
	commandCtx ipfs_oldcmds.Context
}

// PeerHost返回节点的P2P网络主机
// 允许访问底层网络功能
func (im *IpfsMobile) PeerHost() p2p_host.Host {
	return im.IpfsNode.PeerHost
}

// Close关闭IPFS节点并释放资源
func (im *IpfsMobile) Close() error {
	return im.IpfsNode.Close()
}

// ServeCoreHTTP在给定网络监听器上提供IPFS HTTP API服务
// 允许通过HTTP访问IPFS功能
func (im *IpfsMobile) ServeCoreHTTP(l net.Listener, opts ...ipfs_corehttp.ServeOption) error {
	// 配置网关选项，包含WebUI路径
	// 注意：新版API不再需要writable参数
	gatewayOpt := ipfs_corehttp.GatewayOption(ipfs_corehttp.WebUIPaths...)
	// 添加标准选项：WebUI、网关和命令处理
	opts = append(opts,
		ipfs_corehttp.WebUIOption, // 启用Web界面
		gatewayOpt,                // 配置网关
		ipfs_corehttp.CommandsOption(im.commandCtx), // 添加HTTP命令处理
	)

	// 启动HTTP服务
	return ipfs_corehttp.Serve(im.IpfsNode, l, opts...)
}

// ServeGateway在给定网络监听器上提供IPFS HTTP网关服务
// 允许通过HTTP访问IPFS内容
func (im *IpfsMobile) ServeGateway(l net.Listener, writable bool, opts ...ipfs_corehttp.ServeOption) error {
	// 添加标准网关选项
	opts = append(opts,
		ipfs_corehttp.HostnameOption(),                // 处理基于主机名的解析
		ipfs_corehttp.GatewayOption("/ipfs", "/ipns"), // 配置IPFS/IPNS路径
		ipfs_corehttp.VersionOption(),                 // 添加版本信息头
		ipfs_corehttp.CheckVersionOption(),            // 检查客户端兼容性
		// CommandsROOption已被废弃，改用普通的CommandsOption
		ipfs_corehttp.CommandsOption(im.commandCtx), // 命令支持
	)

	// 启动网关服务
	return ipfs_corehttp.Serve(im.IpfsNode, l, opts...)
}

// NewNode根据给定配置创建新的IPFS移动节点
// 这是创建IPFS节点的主要入口点
func NewNode(ctx context.Context, cfg *IpfsConfig) (*IpfsMobile, error) {
	// 预设环境，避免NAT探测和网络发现问题
	os.Setenv("LIBP2P_FORCE_REACHABILITY", "private")
	os.Setenv("LIBP2P_TCP_REUSEPORT", "false")

	// 设置高级别日志记录，确保捕获所有错误
	logging.SetLogLevel("*", "DEBUG")

	// 创建日志目录和文件
	logDir := filepath.Join(cfg.RepoMobile.Path(), "logs")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.Create(filepath.Join(logDir, "ipfs-debug-"+time.Now().Format("20060102-150405")+".log"))
	if err != nil {
		log.Errorf("无法创建日志文件: %s", err)
	} else {
		defer logFile.Close()
		fmt.Fprintf(logFile, "====================== IPFS节点启动日志 ======================\n")
		fmt.Fprintf(logFile, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(logFile, "仓库路径: %s\n", cfg.RepoMobile.Path())
	}

	// 记录初始配置信息
	log.Debugf("开始创建IPFS节点，仓库路径: %s", cfg.RepoMobile.Path())
	if logFile != nil {
		fmt.Fprintf(logFile, "\n[配置信息]\n")
		fmt.Fprintf(logFile, "ExtraOpts: %+v\n", cfg.ExtraOpts)
		fmt.Fprintf(logFile, "HostConfig: %+v\n", cfg.HostConfig)
		fmt.Fprintf(logFile, "RoutingConfig: %+v\n", cfg.RoutingConfig)
	}

	// 填充默认配置值
	log.Debug("填充默认配置")
	if err := cfg.fillDefault(); err != nil {
		log.Errorf("配置无效: %s", err)
		if logFile != nil {
			fmt.Fprintf(logFile, "[错误] 配置无效: %s\n", err)
		}
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// 构建IPFS节点配置
	log.Debug("构建IPFS节点配置")
	if cfg.HostOption == nil {
		cfg.HostOption = SimpleHostOption()
	}
	buildcfg := &ipfs_core.BuildCfg{
		Online:                      true,                                                         // 节点处于在线模式
		Permanent:                   false,                                                        // 非永久节点(适合移动设备)
		DisableEncryptedConnections: false,                                                        // 使用加密连接
		Repo:                        cfg.RepoMobile,                                               // 使用移动仓库
		Host:                        NewHostConfigOption(cfg.HostOption, cfg.HostConfig),          // 配置网络主机
		Routing:                     NewRoutingConfigOption(cfg.RoutingOption, cfg.RoutingConfig), // 配置路由
		ExtraOpts:                   cfg.ExtraOpts,                                                // 设置额外选项(如pubsub)
	}

	if logFile != nil {
		fmt.Fprintf(logFile, "\n[构建配置]\n")
		fmt.Fprintf(logFile, "Online: %t\n", buildcfg.Online)
		fmt.Fprintf(logFile, "Permanent: %t\n", buildcfg.Permanent)
		fmt.Fprintf(logFile, "DisableEncryptedConnections: %t\n", buildcfg.DisableEncryptedConnections)
		fmt.Fprintf(logFile, "ExtraOpts: %+v\n", buildcfg.ExtraOpts)
	}

	// 创建IPFS核心节点
	log.Debug("开始创建IPFS核心节点")
	if logFile != nil {
		fmt.Fprintf(logFile, "\n[创建IPFS核心节点]\n")
		fmt.Fprintf(logFile, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	inode, err := ipfs_core.NewNode(ctx, buildcfg)
	if err != nil {
		log.Errorf("创建IPFS节点失败: %s", err)
		if logFile != nil {
			fmt.Fprintf(logFile, "[严重错误] 创建IPFS节点失败: %s\n", err)
			fmt.Fprintf(logFile, "详细错误信息: %+v\n", err)
			fmt.Fprintf(logFile, "堆栈信息:\n%s\n", debug.Stack())

			// 尝试分析错误链
			type unwrapper interface {
				Unwrap() error
			}

			currentErr := err
			depth := 0
			fmt.Fprintf(logFile, "\n[错误链分析]\n")
			for currentErr != nil && depth < 10 {
				fmt.Fprintf(logFile, "错误层级 %d: %v (%T)\n", depth, currentErr, currentErr)

				if unwrappable, ok := currentErr.(unwrapper); ok {
					currentErr = unwrappable.Unwrap()
				} else {
					currentErr = nil
				}
				depth++
			}
		}
		return nil, fmt.Errorf("failed to init ipfs node: %w", err)
	}

	log.Debug("IPFS核心节点创建成功")
	if logFile != nil {
		fmt.Fprintf(logFile, "[成功] IPFS核心节点创建成功\n")
		fmt.Fprintf(logFile, "节点ID: %s\n", inode.Identity.String())
		fmt.Fprintf(logFile, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	// 创建命令上下文
	log.Debug("创建命令上下文")
	cctx := ipfs_oldcmds.Context{
		ConfigRoot: cfg.RepoMobile.Path(),  // 配置根路径
		ReqLog:     &ipfs_oldcmds.ReqLog{}, // 请求日志
		ConstructNode: func() (*ipfs_core.IpfsNode, error) { // 节点构造函数
			return inode, nil
		},
	}

	log.Debug("IPFS移动节点创建完成")
	// 返回创建的移动IPFS节点
	return &IpfsMobile{
		commandCtx: cctx,           // 命令上下文
		IpfsNode:   inode,          // IPFS核心节点
		Repo:       cfg.RepoMobile, // 仓库引用
	}, nil
}
