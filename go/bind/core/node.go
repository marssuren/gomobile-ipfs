// 文件概览：
// node.go 是 gomobile-ipfs 项目的核心文件，定义了 Node 结构体和相关方法。
// 这个文件主要实现了：
// 1. 创建和初始化 IPFS 节点
// 2. 配置蓝牙传输层
// 3. 管理 mDNS 服务用于本地节点发现
// 4. 提供API和网关服务
// 5. 管理网络连接和资源
// 该文件是移动平台（Android/iOS）访问 IPFS 功能的主要入口点

// ready to use gomobile package for ipfs
// 这是一个可直接用于gomobile的ipfs包

// This package intend to only be use with gomobile bind directly if you
// want to use it in your own gomobile project, you may want to use host/node package directly
// 这个包仅用于直接与gomobile绑定
// 如果你想在自己的gomobile项目中使用，可能需要直接使用host/node包

package core

// Main API exposed to the ios/android
// 暴露给iOS/Android的主要API

import (
	// 导入需要的包
	"context" // 提供上下文控制，用于取消操作和设置超时
	"fmt"     // 格式化输出
	"log"     // 日志功能
	"net"     // 网络操作
	"runtime" // 运行时信息，用于检测平台
	"sync"    // 并发控制

	// 项目内部包
	ble "github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/ble-driver"               // 蓝牙驱动
	ipfs_mobile "github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/ipfsmobile"       // 移动平台IPFS实现
	"github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/ipfsutil"                     // IPFS工具函数
	proximity "github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/proximitytransport" // 近距离传输层
	"go.uber.org/zap"                                                            // 高性能日志库

	// 第三方库
	p2p_mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns" // mDNS服务发现
	ma "github.com/multiformats/go-multiaddr"                 // 多地址处理
	manet "github.com/multiformats/go-multiaddr/net"          // 多地址网络接口

	// IPFS核心组件
	ipfs_bs "github.com/ipfs/boxo/bootstrap"  // IPFS引导节点
	ipfs_config "github.com/ipfs/kubo/config" // IPFS配置
	libp2p "github.com/libp2p/go-libp2p"      // P2P网络库
)

// Node 结构体定义，代表一个IPFS节点
type Node struct {
	listeners   []manet.Listener // 网络监听器列表
	muListeners sync.Mutex       // 保护listeners的互斥锁
	mdnsLocker  sync.Locker      // mDNS锁，控制mDNS服务的访问
	mdnsLocked  bool             // 标记mDNS是否被锁定
	mdnsService p2p_mdns.Service // mDNS服务，用于本地网络发现

	ipfsMobile *ipfs_mobile.IpfsMobile // 移动平台IPFS节点实例
}

// 检测当前平台是否为Android
func isAndroidPlatform() bool {
	return runtime.GOOS == "android"
}

// 检测当前网络环境是否受限
func isNetworkLimited() bool {
	// 尝试使用netlink API，如果失败则认为网络环境受限
	_, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("网络接口检测失败: %v", err)
		return true
	}

	// 如果是Android平台，认为网络环境受限
	if isAndroidPlatform() {
		log.Printf("检测到Android平台，使用受限网络配置")
		return true
	}

	return false
}

// NewNode 创建一个新的IPFS节点
func NewNode(r *Repo, config *NodeConfig) (*Node, error) {
	// 如果没有提供配置，创建默认配置
	if config == nil {
		config = NewNodeConfig()
	}

	// 设置DNS解析器，使用固定的DNS服务器
	var dialer net.Dialer
	net.DefaultResolver = &net.Resolver{
		PreferGo: false, // 不使用Go的DNS解析器
		Dial: func(context context.Context, _, _ string) (net.Conn, error) {
			// 使用硬编码的DNS服务器(84.200.69.80是privacy-friendly的DNS服务器)
			conn, err := dialer.DialContext(context, "udp", "84.200.69.80:53")
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}

	// 创建上下文
	ctx := context.Background()

	// 加载IPFS插件
	if _, err := loadPlugins(r.mr.Path()); err != nil {
		return nil, err
	}

	// 设置自定义网络驱动（如果提供）
	if config.netDriver != nil {
		logger, _ := zap.NewDevelopment()
		inet := &inet{
			net:    config.netDriver,
			logger: logger,
		}
		// 配置自定义网络接口
		ipfsutil.SetNetDriver(inet)
	}

	// 蓝牙选项变量
	var bleOpt libp2p.Option

	// 根据平台选择不同的蓝牙驱动实现
	switch {
	// Java嵌入式驱动（Android平台）
	case config.bleDriver != nil:
		logger := zap.NewExample()
		defer func() {
			if err := logger.Sync(); err != nil {
				fmt.Println(err)
			}
		}()
		// 使用传入的蓝牙驱动创建传输层
		bleOpt = libp2p.Transport(proximity.NewTransport(ctx, logger, config.bleDriver))
	// Go嵌入式驱动（iOS平台）
	case ble.Supported:
		logger := zap.NewExample()
		defer func() {
			if err := logger.Sync(); err != nil {
				fmt.Println(err)
			}
		}()
		// 创建并使用iOS蓝牙驱动
		bleOpt = libp2p.Transport(proximity.NewTransport(ctx, logger, ble.NewDriver(logger)))
	default:
		// 如果平台不支持蓝牙，输出日志
		log.Printf("cannot enable BLE on an unsupported platform")
	}

	// 检测网络环境，并选择合适的配置
	networkLimited := isNetworkLimited()

	// 配置IPFS节点
	ipfscfg := &ipfs_mobile.IpfsConfig{
		HostConfig: &ipfs_mobile.HostConfig{
			Options: []libp2p.Option{
				bleOpt,
				libp2p.DisableRelay(),             // 禁用中继功能
				libp2p.ForceReachabilityPrivate(), // 强制私有网络，避免NAT检测
			},
		},
		RepoMobile: r.mr,
	}

	// 如果是受限网络环境（如Android），使用SimpleHostOption绕过循环依赖
	if networkLimited {
		log.Printf("检测到受限网络环境，使用SimpleHostOption和精简配置")
		ipfscfg.HostOption = ipfs_mobile.SimpleHostOption()
		ipfscfg.ExtraOpts = map[string]bool{
			"pubsub":    false, // 禁用pubsub，避免额外的网络复杂性
			"ipnsps":    false, // 禁用IPNS over pubsub
			"dht":       false, // 禁用完整DHT
			"dhtclient": true,  // 使用客户端模式DHT
		}
	} else {
		// 正常网络环境，使用完整功能
		ipfscfg.ExtraOpts = map[string]bool{
			"pubsub": true, // 启用实验性的pubsub功能
			"ipnsps": true, // 启用IPNS over pubsub
		}
	}

	// 获取仓库配置
	cfg, err := r.mr.Config()
	if err != nil {
		panic(err)
	}

	// mDNS处理（多播DNS，用于本地网络发现）
	mdnsLocked := false
	if cfg.Discovery.MDNS.Enabled && config.mdnsLockerDriver != nil {
		// 锁定mDNS（避免多个进程同时使用）
		config.mdnsLockerDriver.Lock()
		mdnsLocked = true

		// 暂时禁用mDNS，避免ipfs_mobile.NewNode启动它
		err := r.mr.ApplyPatchs(func(cfg *ipfs_config.Config) error {
			cfg.Discovery.MDNS.Enabled = false
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("unable to ApplyPatchs to disable mDNS: %w", err)
		}
	}

	// 创建移动IPFS节点
	mnode, err := ipfs_mobile.NewNode(ctx, ipfscfg)
	if err != nil {
		// 如果mDNS已锁定但创建节点失败，释放锁
		if mdnsLocked {
			config.mdnsLockerDriver.Unlock()
		}
		return nil, err
	}

	// mDNS服务变量
	var mdnsService p2p_mdns.Service = nil
	if mdnsLocked {
		// 恢复mDNS配置
		err := r.mr.ApplyPatchs(func(cfg *ipfs_config.Config) error {
			cfg.Discovery.MDNS.Enabled = true
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("unable to ApplyPatchs to enable mDNS: %w", err)
		}

		// 获取对等节点主机
		h := mnode.PeerHost()
		mdnslogger, _ := zap.NewDevelopment()

		// 创建发现处理器和mDNS服务
		dh := ipfsutil.DiscoveryHandler(ctx, mdnslogger, h)
		mdnsService = ipfsutil.NewMdnsService(mdnslogger, h, ipfsutil.MDNSServiceName, dh)

		// 启动mDNS服务
		// 获取多播接口
		ifaces, err := ipfsutil.GetMulticastInterfaces()
		if err != nil {
			if mdnsLocked {
				config.mdnsLockerDriver.Unlock()
			}
			return nil, err
		}

		// 如果找到多播接口，启动mDNS服务
		if len(ifaces) > 0 {
			mdnslogger.Info("starting mdns")
			if err := mdnsService.Start(); err != nil {
				if mdnsLocked {
					config.mdnsLockerDriver.Unlock()
				}
				return nil, fmt.Errorf("unable to start mdns service: %w", err)
			}
		} else {
			mdnslogger.Error("unable to start mdns service, no multicast interfaces found")
		}
	}

	// 使用默认配置引导节点
	if err := mnode.IpfsNode.Bootstrap(ipfs_bs.DefaultBootstrapConfig); err != nil {
		log.Printf("failed to bootstrap node: `%s`", err)
	}

	// 返回创建的节点
	return &Node{
		ipfsMobile:  mnode,
		mdnsLocker:  config.mdnsLockerDriver,
		mdnsLocked:  mdnsLocked,
		mdnsService: mdnsService,
	}, nil
}

// Close 关闭节点并释放资源
func (n *Node) Close() error {
	// 关闭所有监听器
	n.muListeners.Lock()
	for _, l := range n.listeners {
		l.Close()
	}
	n.muListeners.Unlock()

	// 如果mDNS已锁定，关闭服务并释放锁
	if n.mdnsLocked {
		n.mdnsService.Close()
		n.mdnsLocker.Unlock()
		n.mdnsLocked = false
	}

	// 关闭IPFS节点
	return n.ipfsMobile.Close()
}

// ServeUnixSocketAPI 在Unix套接字上提供API服务
func (n *Node) ServeUnixSocketAPI(sockpath string) (err error) {
	_, err = n.ServeAPIMultiaddr("/unix/" + sockpath)
	return
}

// ServeTCPAPI 在指定端口上提供TCP API服务，并返回监听地址
func (n *Node) ServeTCPAPI(port string) (string, error) {
	return n.ServeAPIMultiaddr("/ip4/127.0.0.1/tcp/" + port)
}

// ServeConfig 根据配置提供API和网关服务
func (n *Node) ServeConfig() error {
	// 获取配置
	cfg, err := n.ipfsMobile.Repo.Config()
	if err != nil {
		return fmt.Errorf("unable to get config: %s", err.Error())
	}

	// 启动所有配置的API服务
	if len(cfg.Addresses.API) > 0 {
		for _, maddr := range cfg.Addresses.API {
			if _, err := n.ServeAPIMultiaddr(maddr); err != nil {
				return fmt.Errorf("cannot serve `%s`: %s", maddr, err.Error())
			}
		}
	}

	// 启动所有配置的网关服务（默认只读）
	if len(cfg.Addresses.Gateway) > 0 {
		for _, maddr := range cfg.Addresses.Gateway {
			// 公共网关默认为只读
			if _, err := n.ServeGatewayMultiaddr(maddr, false); err != nil {
				return fmt.Errorf("cannot serve `%s`: %s", maddr, err.Error())
			}
		}
	}

	return nil
}

// ServeUnixSocketGateway 在Unix套接字上提供网关服务
func (n *Node) ServeUnixSocketGateway(sockpath string, writable bool) (err error) {
	_, err = n.ServeGatewayMultiaddr("/unix/"+sockpath, writable)
	return
}

// ServeTCPGateway 在指定端口上提供TCP网关服务
func (n *Node) ServeTCPGateway(port string, writable bool) (string, error) {
	return n.ServeGatewayMultiaddr("/ip4/127.0.0.1/tcp/"+port, writable)
}

// ServeGatewayMultiaddr 在指定多地址上提供网关服务
func (n *Node) ServeGatewayMultiaddr(smaddr string, writable bool) (string, error) {
	// 解析多地址
	maddr, err := ma.NewMultiaddr(smaddr)
	if err != nil {
		return "", err
	}

	// 在该地址上监听
	ml, err := manet.Listen(maddr)
	if err != nil {
		return "", err
	}

	// 保存监听器
	n.muListeners.Lock()
	n.listeners = append(n.listeners, ml)
	n.muListeners.Unlock()

	// 启动网关服务（在新协程中）
	go func(l net.Listener) {
		if err := n.ipfsMobile.ServeGateway(l, writable); err != nil {
			log.Printf("serve error: %s", err.Error())
		}
	}(manet.NetListener(ml))

	// 返回实际监听的地址
	return ml.Multiaddr().String(), nil
}

// ServeAPIMultiaddr 在指定多地址上提供API服务
func (n *Node) ServeAPIMultiaddr(smaddr string) (string, error) {
	// 解析多地址
	maddr, err := ma.NewMultiaddr(smaddr)
	if err != nil {
		return "", err
	}

	// 在该地址上监听
	ml, err := manet.Listen(maddr)
	if err != nil {
		return "", err
	}

	// 保存监听器
	n.muListeners.Lock()
	n.listeners = append(n.listeners, ml)
	n.muListeners.Unlock()

	// 启动API服务（在新协程中）
	go func(l net.Listener) {
		if err := n.ipfsMobile.ServeCoreHTTP(l); err != nil {
			log.Printf("serve error: %s", err.Error())
		}
	}(manet.NetListener(ml))

	// 返回实际监听的地址
	return ml.Multiaddr().String(), nil
}

// init 是Go的特殊函数，在包初始化时自动执行
func init() {
	// 以下代码被注释掉了，不会执行
	// ipfs_log.SetDebugLogging() // 设置调试日志
}
