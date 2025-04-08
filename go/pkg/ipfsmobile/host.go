/*
文件概览：go/pkg/ipfsmobile/host.go
这个文件定义了移动平台IPFS节点的网络主机(Host)实现。主要功能和组件包括：
1. 定义HostMobile结构，封装libp2p网络主机
2. 提供灵活的主机配置机制，支持链式配置和选项组合
3. 实现自定义主机选项，允许添加移动平台特定的网络功能(如蓝牙传输)
4. 集成IPFS的P2P网络架构与移动平台优化

这个文件是gomobile-ipfs中网络层的核心，负责处理P2P连接、传输协议和网络发现，
并提供了适合移动环境的网络配置能力。
*/

// node包提供移动平台IPFS节点实现
package node

import (
	"fmt"

	// libp2p核心库
	p2p "github.com/libp2p/go-libp2p"                       // libp2p网络库主包
	p2p_host "github.com/libp2p/go-libp2p/core/host"        // 网络主机接口
	p2p_peer "github.com/libp2p/go-libp2p/core/peer"        // 对等节点标识
	p2p_pstore "github.com/libp2p/go-libp2p/core/peerstore" // 对等节点存储

	// IPFS网络库
	ipfs_p2p "github.com/ipfs/kubo/core/node/libp2p" // IPFS的libp2p网络配置
)

// 类型检查断言：确保HostMobile实现了p2p_host.Host接口
// 这是Go中验证接口实现的标准方式
var _ p2p_host.Host = (*HostMobile)(nil)

// HostConfigFunc定义一个函数类型，用于配置主机
// 它接收创建好的主机实例，可以对其进行配置，并返回可能的错误
type HostConfigFunc func(p2p_host.Host) error

// HostConfig定义主机的配置选项
// @TODO: 注释表明这里计划添加更多移动平台特定的选项
type HostConfig struct {
	// 主机初始化后调用的配置函数
	ConfigFunc HostConfigFunc

	// libp2p网络选项列表，可以包含传输协议、安全选项等
	Options []p2p.Option
}

// ChainHostConfig将多个主机配置函数链接在一起
// 类似于之前看到的配置补丁链接模式
// 参数:
//
//	cfgs: 要链接的主机配置函数列表
//
// 返回:
//
//	一个新的配置函数，它会顺序应用所有配置
func ChainHostConfig(cfgs ...HostConfigFunc) HostConfigFunc {
	return func(host p2p_host.Host) (err error) {
		// 遍历所有配置函数
		for _, cfg := range cfgs {
			// 跳过空配置
			if cfg == nil {
				continue // skip empty config
			}

			// 应用当前配置，出错则返回
			if err = cfg(host); err != nil {
				return
			}
		}
		return
	}
}

// HostMobile是p2p主机的移动平台封装
// 它嵌入了标准libp2p主机接口，继承其所有方法
type HostMobile struct {
	p2p_host.Host // 嵌入主机接口，继承其方法
}

// NewHostConfigOption创建一个新的IPFS主机配置选项
// 这个函数接合了IPFS的主机选项系统和我们自定义的HostConfig
// 参数:
//
//	hopt: 基础IPFS主机选项
//	cfg: 自定义主机配置
//
// 返回:
//
//	一个新的IPFS主机选项函数，集成了自定义配置
func NewHostConfigOption(hopt ipfs_p2p.HostOption, cfg *HostConfig) ipfs_p2p.HostOption {
	// 返回符合IPFS主机选项接口的函数
	return func(id p2p_peer.ID, ps p2p_pstore.Peerstore, options ...p2p.Option) (p2p_host.Host, error) {
		// 添加自定义P2P选项
		if cfg.Options != nil {
			options = append(options, cfg.Options...)
		}

		// 使用基础选项创建主机
		host, err := hopt(id, ps, options...)
		if err != nil {
			return nil, err
		}

		// 如果提供了配置函数，应用它
		if cfg.ConfigFunc != nil {
			// 应用自定义主机配置
			if err := cfg.ConfigFunc(host); err != nil {
				return nil, fmt.Errorf("unable to apply host config: %w", err)
			}
		}

		// 返回配置好的主机
		return host, nil
	}
}
