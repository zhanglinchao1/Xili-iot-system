package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresTestContainer PostgreSQL测试容器
type PostgresTestContainer struct {
	Container  testcontainers.Container
	ConnString string
	Host       string
	Port       string
}

// NewPostgresTestContainer 创建PostgreSQL测试容器
// withTimescale: 是否使用TimescaleDB镜像
func NewPostgresTestContainer(ctx context.Context, withTimescale bool) (*PostgresTestContainer, error) {
	image := "postgres:14-alpine"
	if withTimescale {
		image = "timescale/timescaledb:latest-pg14"
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
			"POSTGRES_DB":       "test_db",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	port := mappedPort.Port()
	connString := fmt.Sprintf("host=%s port=%s user=test_user password=test_password dbname=test_db sslmode=disable",
		host, port)

	return &PostgresTestContainer{
		Container:  container,
		ConnString: connString,
		Host:       host,
		Port:       port,
	}, nil
}

// Close 关闭并删除测试容器
func (p *PostgresTestContainer) Close(ctx context.Context) error {
	if p.Container != nil {
		return p.Container.Terminate(ctx)
	}
	return nil
}

// GetConnectionString 获取连接字符串
func (p *PostgresTestContainer) GetConnectionString() string {
	return p.ConnString
}
