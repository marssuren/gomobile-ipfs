/*
文件概览：go/pkg/ipfsmobile/repo.go
这个文件定义了移动平台IPFS仓库(Repository)的实现。主要功能和组件包括：
1. 定义RepoMobile结构，作为标准IPFS仓库的移动平台适配器
2. 提供配置补丁(patch)机制，允许动态修改IPFS配置
3. 实现配置链式修改功能，支持多个配置变更的组合应用
4. 确保移动平台仓库实现符合IPFS仓库接口规范

这个文件是gomobile-ipfs中仓库层的核心，它连接了IPFS的存储系统和移动应用，
提供了适合移动环境的仓库管理和配置能力。
*/

// node包提供移动平台IPFS节点和仓库的核心实现
package node

import (
	// 导入IPFS核心库
	ipfs_config "github.com/ipfs/kubo/config" // IPFS配置系统
	ipfs_repo "github.com/ipfs/kubo/repo"     // IPFS仓库接口
)

// 类型检查断言：确保RepoMobile实现了ipfs_repo.Repo接口
// 这是Go中验证接口实现的常用模式
var _ ipfs_repo.Repo = (*RepoMobile)(nil)

// RepoConfigPatch定义一个函数类型，用于修改IPFS配置
// 每个补丁函数接收一个配置指针并可以修改它，返回可能的错误
type RepoConfigPatch func(cfg *ipfs_config.Config) (err error)

// RepoMobile是标准IPFS仓库的移动平台封装
// 它添加了路径信息和配置补丁功能
type RepoMobile struct {
	// 嵌入标准IPFS仓库接口，继承其所有方法
	ipfs_repo.Repo

	// 仓库在文件系统中的路径
	// 在移动环境中，这通常指向应用数据目录
	Path string
}

// NewRepoMobile创建一个新的移动平台仓库实例
// 参数:
//
//	path: 仓库在文件系统中的路径
//	repo: 底层IPFS仓库实现
//
// 返回:
//
//	移动平台仓库实例
func NewRepoMobile(path string, repo ipfs_repo.Repo) *RepoMobile {
	return &RepoMobile{
		Repo: repo, // 存储底层仓库实现
		Path: path, // 保存仓库路径
	}
}

// ApplyPatchs应用一系列配置补丁到仓库配置
// 这允许以可组合的方式修改IPFS配置
// 参数:
//
//	patchs: 要应用的配置补丁函数变参
//
// 返回:
//
//	可能的错误
func (mr *RepoMobile) ApplyPatchs(patchs ...RepoConfigPatch) error {
	// 获取当前配置
	cfg, err := mr.Config()
	if err != nil {
		return err
	}

	// 使用链式补丁函数应用所有补丁
	if err := ChainIpfsConfigPatch(patchs...)(cfg); err != nil {
		return err
	}

	// 将修改后的配置保存回仓库
	return mr.SetConfig(cfg)
}

// ChainIpfsConfigPatch将多个配置补丁函数合并为一个
// 这是函数式编程中的组合模式
// 参数:
//
//	patchs: 要链接的配置补丁函数变参
//
// 返回:
//
//	合并后的配置补丁函数
func ChainIpfsConfigPatch(patchs ...RepoConfigPatch) RepoConfigPatch {
	// 返回一个新函数，它将依次应用所有补丁
	return func(cfg *ipfs_config.Config) (err error) {
		// 遍历所有补丁函数
		for _, patch := range patchs {
			// 跳过空补丁
			if patch == nil {
				continue // skip empty patch
			}

			// 应用当前补丁，如果出错则返回
			if err = patch(cfg); err != nil {
				return
			}
		}
		// 所有补丁应用成功
		return
	}
}
