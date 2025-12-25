package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud-system/internal/abac"
	"cloud-system/internal/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// PolicyPublisher MQTT策略发布器
type PolicyPublisher struct {
	client     mqtt.Client
	policyRepo abac.PolicyRepository
}

// NewPolicyPublisher 创建策略发布器
func NewPolicyPublisher(client mqtt.Client, policyRepo abac.PolicyRepository) *PolicyPublisher {
	return &PolicyPublisher{
		client:     client,
		policyRepo: policyRepo,
	}
}

// PolicySyncMessage 策略同步消息格式
type PolicySyncMessage struct {
	Action    string               `json:"action"` // sync, delete, full_sync
	Policies  []*abac.AccessPolicy `json:"policies,omitempty"`
	PolicyIDs []string             `json:"policy_ids,omitempty"`
	Timestamp time.Time            `json:"timestamp"`
}

// DistributePolicyToCabinet 分发策略到指定储能柜
func (p *PolicyPublisher) DistributePolicyToCabinet(ctx context.Context, cabinetID string, policyID string) error {
	policy, err := p.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return fmt.Errorf("获取策略失败: %w", err)
	}

	// 只分发device类型策略
	if policy.SubjectType != "device" {
		return fmt.Errorf("只能分发device类型策略到储能柜")
	}

	msg := PolicySyncMessage{
		Action:    "sync",
		Policies:  []*abac.AccessPolicy{policy},
		Timestamp: time.Now(),
	}

	return p.publish(cabinetID, msg)
}

// DistributePoliciesToCabinet 分发多个策略到指定储能柜
func (p *PolicyPublisher) DistributePoliciesToCabinet(ctx context.Context, cabinetID string, policyIDs []string) error {
	var policies []*abac.AccessPolicy

	for _, id := range policyIDs {
		policy, err := p.policyRepo.GetByID(ctx, id)
		if err != nil {
			utils.Warn("获取策略失败", zap.String("policy_id", id), zap.Error(err))
			continue
		}
		if policy.SubjectType == "device" {
			policies = append(policies, policy)
		}
	}

	if len(policies) == 0 {
		return fmt.Errorf("没有可分发的device类型策略")
	}

	msg := PolicySyncMessage{
		Action:    "sync",
		Policies:  policies,
		Timestamp: time.Now(),
	}

	return p.publish(cabinetID, msg)
}

// FullSyncToCabinet 全量同步所有device策略到指定储能柜
func (p *PolicyPublisher) FullSyncToCabinet(ctx context.Context, cabinetID string) error {
	policies, err := p.policyRepo.GetBySubjectType(ctx, "device", false)
	if err != nil {
		return fmt.Errorf("获取device策略失败: %w", err)
	}

	msg := PolicySyncMessage{
		Action:    "full_sync",
		Policies:  policies,
		Timestamp: time.Now(),
	}

	return p.publish(cabinetID, msg)
}

// DeletePolicyFromCabinet 从指定储能柜删除策略
func (p *PolicyPublisher) DeletePolicyFromCabinet(ctx context.Context, cabinetID string, policyIDs []string) error {
	msg := PolicySyncMessage{
		Action:    "delete",
		PolicyIDs: policyIDs,
		Timestamp: time.Now(),
	}

	return p.publish(cabinetID, msg)
}

// BroadcastPolicyToAllCabinets 广播策略到所有储能柜
func (p *PolicyPublisher) BroadcastPolicyToAllCabinets(ctx context.Context, policyID string) error {
	policy, err := p.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return fmt.Errorf("获取策略失败: %w", err)
	}

	if policy.SubjectType != "device" {
		return fmt.Errorf("只能广播device类型策略")
	}

	msg := PolicySyncMessage{
		Action:    "sync",
		Policies:  []*abac.AccessPolicy{policy},
		Timestamp: time.Now(),
	}

	// 使用通配符主题广播
	topic := "cloud/cabinet/+/policy/sync"
	return p.publishToTopic(topic, msg)
}

func (p *PolicyPublisher) publish(cabinetID string, msg PolicySyncMessage) error {
	topic := fmt.Sprintf("cloud/cabinet/%s/policy/sync", cabinetID)
	return p.publishToTopic(topic, msg)
}

func (p *PolicyPublisher) publishToTopic(topic string, msg PolicySyncMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	token := p.client.Publish(topic, 1, false, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("发布MQTT消息失败: %w", token.Error())
	}

	utils.Info("策略同步消息已发送",
		zap.String("topic", topic),
		zap.String("action", msg.Action),
		zap.Int("policy_count", len(msg.Policies)),
	)

	return nil
}
