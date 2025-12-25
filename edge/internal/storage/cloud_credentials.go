package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// CloudCredential Cloud端API凭证
type CloudCredential struct {
	ID            int       `db:"id" json:"id"`
	CabinetID     string    `db:"cabinet_id" json:"cabinet_id"`
	APIKey        string    `db:"api_key" json:"api_key"`
	APISecret     string    `db:"api_secret" json:"api_secret"`           // 已废弃,保留用于兼容
	CloudEndpoint string    `db:"cloud_endpoint" json:"cloud_endpoint"`
	Enabled       bool      `db:"enabled" json:"enabled"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// SaveCloudCredentials 保存或更新Cloud端凭证
func (s *SQLiteDB) SaveCloudCredentials(cabinetID, apiKey, apiSecret, cloudEndpoint string) error {
	query := `
		INSERT INTO cloud_credentials (cabinet_id, api_key, api_secret, cloud_endpoint, enabled)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(cabinet_id) DO UPDATE SET
			api_key = excluded.api_key,
			api_secret = excluded.api_secret,
			cloud_endpoint = excluded.cloud_endpoint,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.Exec(query, cabinetID, apiKey, apiSecret, cloudEndpoint)
	if err != nil {
		return fmt.Errorf("保存Cloud凭证失败: %w", err)
	}

	return nil
}

// GetCloudCredentials 获取Cloud端凭证
func (s *SQLiteDB) GetCloudCredentials(cabinetID string) (*CloudCredential, error) {
	var cred CloudCredential
	query := `SELECT id, cabinet_id, api_key, api_secret, cloud_endpoint, enabled, created_at, updated_at
	          FROM cloud_credentials WHERE cabinet_id = ? AND enabled = 1 LIMIT 1`

	err := s.db.QueryRow(query, cabinetID).Scan(
		&cred.ID, &cred.CabinetID, &cred.APIKey, &cred.APISecret,
		&cred.CloudEndpoint, &cred.Enabled, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // 返回nil表示没有凭证
	}
	if err != nil {
		return nil, fmt.Errorf("获取Cloud凭证失败: %w", err)
	}

	return &cred, nil
}

// GetFirstCloudCredentials 获取第一个启用的凭证(用于单储能柜场景)
func (s *SQLiteDB) GetFirstCloudCredentials() (*CloudCredential, error) {
	var cred CloudCredential
	query := `SELECT id, cabinet_id, api_key, api_secret, cloud_endpoint, enabled, created_at, updated_at
	          FROM cloud_credentials WHERE enabled = 1 ORDER BY created_at ASC LIMIT 1`

	err := s.db.QueryRow(query).Scan(
		&cred.ID, &cred.CabinetID, &cred.APIKey, &cred.APISecret,
		&cred.CloudEndpoint, &cred.Enabled, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // 返回nil表示没有凭证
	}
	if err != nil {
		return nil, fmt.Errorf("获取Cloud凭证失败: %w", err)
	}

	return &cred, nil
}

// DeleteCloudCredentials 删除凭证
func (s *SQLiteDB) DeleteCloudCredentials(cabinetID string) error {
	query := `DELETE FROM cloud_credentials WHERE cabinet_id = ?`

	_, err := s.db.Exec(query, cabinetID)
	if err != nil {
		return fmt.Errorf("删除Cloud凭证失败: %w", err)
	}

	return nil
}

// DisableCloudCredentials 禁用凭证(软删除)
func (s *SQLiteDB) DisableCloudCredentials(cabinetID string) error {
	query := `UPDATE cloud_credentials SET enabled = 0, updated_at = CURRENT_TIMESTAMP WHERE cabinet_id = ?`

	_, err := s.db.Exec(query, cabinetID)
	if err != nil {
		return fmt.Errorf("禁用Cloud凭证失败: %w", err)
	}

	return nil
}

// ListCloudCredentials 列出所有凭证
func (s *SQLiteDB) ListCloudCredentials() ([]*CloudCredential, error) {
	var creds []*CloudCredential
	query := `SELECT id, cabinet_id, api_key, api_secret, cloud_endpoint, enabled, created_at, updated_at
	          FROM cloud_credentials ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("列出Cloud凭证失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cred CloudCredential
		if err := rows.Scan(
			&cred.ID, &cred.CabinetID, &cred.APIKey, &cred.APISecret,
			&cred.CloudEndpoint, &cred.Enabled, &cred.CreatedAt, &cred.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描Cloud凭证失败: %w", err)
		}
		creds = append(creds, &cred)
	}

	return creds, nil
}
