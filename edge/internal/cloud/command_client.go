package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/edge/storage-cabinet/internal/config"
	"go.uber.org/zap"
)

// CommandClient 调用Cloud命令回执接口
type CommandClient struct {
	cfg        config.CloudConfig
	logger     *zap.Logger
	httpClient *http.Client
}

// NewCommandClient 创建命令客户端
func NewCommandClient(cfg config.CloudConfig, logger *zap.Logger) *CommandClient {
	return &CommandClient{
		cfg:    cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AckCommand 回执命令状态
func (c *CommandClient) AckCommand(commandID, status, message string) error {
	if !c.cfg.Enabled || c.cfg.Endpoint == "" {
		return fmt.Errorf("cloud config disabled")
	}

	baseURL := strings.TrimSuffix(c.cfg.Endpoint, "/")
	url := fmt.Sprintf("%s/commands/%s/ack", baseURL, commandID)

	payload := map[string]string{
		"status": status,
	}
	if message != "" {
		payload["message"] = message
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("cloud ack failed: status %d", resp.StatusCode)
	}

	return nil
}
