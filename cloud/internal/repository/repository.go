package repository

import "context"

// Repository 基础仓库接口
type Repository interface {
	// Close 关闭连接
	Close() error
	// Ping 检查连接
	Ping(ctx context.Context) error
}

// Transactional 事务接口
type Transactional interface {
	// BeginTx 开始事务
	BeginTx(ctx context.Context) (interface{}, error)
	// CommitTx 提交事务
	CommitTx(tx interface{}) error
	// RollbackTx 回滚事务
	RollbackTx(tx interface{}) error
}
