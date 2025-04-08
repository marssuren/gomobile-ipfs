/*
文件概览：go/pkg/ipfsmobile/routing.go
此文件定义了移动平台IPFS节点的内容路由(Routing)配置系统。主要功能：
1. 提供自定义路由配置的接口和实现
2. 将标准IPFS路由系统与移动平台特定需求集成
3. 支持灵活配置DHT、内容路由策略等网络发现功能
4. 与host.go文件设计模式一致，采用装饰器和函数选项模式

路由系统负责在IPFS网络中定位内容和节点，对移动端的网络效率和电池使用有重要影响。
*/

// 与host.go在同一个包中
package node

import (
	"context"
	"fmt"

	ds "github.com/ipfs/go-datastore"                      // IPFS数据存储接口
	ipfs_p2p "github.com/ipfs/kubo/core/node/libp2p"       // IPFS的libp2p实现
	p2p_record "github.com/libp2p/go-libp2p-record"        // libp2p记录验证
	p2p_host "github.com/libp2p/go-libp2p/core/host"       // libp2p主机接口
	p2p_peer "github.com/libp2p/go-libp2p/core/peer"       // 对等节点标识
	p2p_routing "github.com/libp2p/go-libp2p/core/routing" // 内容路由接口
)

// RoutingConfigFunc定义配置路由系统的函数类型
// 接收主机和路由实例，可对路由进行配置，返回可能的错误
type RoutingConfigFunc func(p2p_host.Host, p2p_routing.Routing) error

// RoutingConfig定义路由系统的配置选项
// 与Host配置结构相似，但专注于路由系统
type RoutingConfig struct {
	ConfigFunc RoutingConfigFunc // 路由配置函数
}

// NewRoutingConfigOption创建新的IPFS路由配置选项
// 将自定义路由配置与IPFS标准路由系统集成
// 参数:
//
//	ro: 基础IPFS路由选项
//	rc: 自定义路由配置
//
// 返回:
//
//	集成了自定义配置的IPFS路由选项函数
func NewRoutingConfigOption(ro ipfs_p2p.RoutingOption, rc *RoutingConfig) ipfs_p2p.RoutingOption {
	// 返回符合IPFS路由选项接口的函数
	return func(
		ctx context.Context, // 上下文，用于取消操作
		host p2p_host.Host, // 网络主机实例
		dstore ds.Batching, // 数据存储实例
		validator p2p_record.Validator, // 记录验证器
		bootstrapPeers ...p2p_peer.AddrInfo, // 启动节点信息
	) (p2p_routing.Routing, error) {
		// 使用基础选项创建路由系统
		routing, err := ro(ctx, host, dstore, validator, bootstrapPeers...)
		if err != nil {
			return nil, err
		}

		// 如果提供了配置函数，应用它
		if rc.ConfigFunc != nil {
			if err := rc.ConfigFunc(host, routing); err != nil {
				return nil, fmt.Errorf("failed to config routing: %w", err)
			}
		}

		// 返回配置好的路由系统
		return routing, nil
	}
}
