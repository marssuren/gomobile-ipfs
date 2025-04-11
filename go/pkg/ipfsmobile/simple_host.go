package node

import (
	"fmt"

	"github.com/ipfs/kubo/core/node/libp2p"
	p2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

// 简化版主机选项，完全绕过fx依赖注入
func SimpleHostOption() libp2p.HostOption {
	return func(id peer.ID, ps peerstore.Peerstore, options ...p2p.Option) (host.Host, error) {
		// 预先设置环境变量，强制某些行为
		// os.Setenv("LIBP2P_FORCE_REACHABILITY", "private")

		// 获取私钥
		pkey := ps.PrivKey(id)
		if pkey == nil {
			return nil, fmt.Errorf("missing private key for node ID: %s", id)
		}

		// 基础选项
		baseOpts := []p2p.Option{
			p2p.Identity(pkey),
			p2p.Peerstore(ps),
			p2p.NoListenAddrs,
			p2p.EnableRelay(), // 启用Relay功能是使用AutoRelay的前提
			p2p.ForceReachabilityPrivate(),
		}

		// 合并传入的选项和基础选项
		allOpts := append(baseOpts, options...)

		// 直接创建libp2p主机，完全绕过fx
		return p2p.New(allOpts...)
	}
}
