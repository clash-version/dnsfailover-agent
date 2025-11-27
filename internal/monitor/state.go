package monitor

import (
	"sync"
	"time"
)

// DomainState 域名运行时状态（仅存在于内存中）
type DomainState struct {
	Domain            string    // 域名
	FailCount         int       // 当前周期内的连续失败次数
	LastSwitchTime    time.Time // 最后一次切换时间
	SwitchCooldownMin int       // 切换冷却期（分钟）
}

// StateManager 状态管理器（内存中维护域名状态）
type StateManager struct {
	states map[string]*DomainState
	mu     sync.RWMutex
}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]*DomainState),
	}
}

// InitDomain 初始化域名状态（启动监控时调用，失败计数设为0）
func (sm *StateManager) InitDomain(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.states[domain] = &DomainState{
		Domain:            domain,
		FailCount:         0,
		LastSwitchTime:    time.Time{}, // 零值表示从未切换过
		SwitchCooldownMin: 5,           // 默认5分钟冷却期
	}
}

// GetFailCount 获取域名的失败计数
func (sm *StateManager) GetFailCount(domain string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[domain]; exists {
		return state.FailCount
	}
	return 0
}

// IncrementFailCount 增加失败计数
func (sm *StateManager) IncrementFailCount(domain string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.FailCount++
		return state.FailCount
	}

	// 如果不存在，创建新状态
	sm.states[domain] = &DomainState{
		Domain:            domain,
		FailCount:         1,
		LastSwitchTime:    time.Time{},
		SwitchCooldownMin: 5,
	}
	return 1
}

// ResetFailCount 重置失败计数为0
func (sm *StateManager) ResetFailCount(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.FailCount = 0
	}
}

// RemoveDomain 移除域名状态（当域名被删除时调用）
func (sm *StateManager) RemoveDomain(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, domain)
}

// GetAllStates 获取所有域名状态（用于调试）
func (sm *StateManager) GetAllStates() map[string]*DomainState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make(map[string]*DomainState)
	for k, v := range sm.states {
		result[k] = &DomainState{
			Domain:            v.Domain,
			FailCount:         v.FailCount,
			LastSwitchTime:    v.LastSwitchTime,
			SwitchCooldownMin: v.SwitchCooldownMin,
		}
	}
	return result
}

// MarkSwitched 标记域名已执行切换（记录切换时间）
func (sm *StateManager) MarkSwitched(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.LastSwitchTime = time.Now()
		state.FailCount = 0 // 切换后重置失败计数
	}
}

// IsInCooldown 检查域名是否在冷却期内
func (sm *StateManager) IsInCooldown(domain string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[domain]
	if !exists {
		return false
	}

	// 如果从未切换过（零值时间），不在冷却期
	if state.LastSwitchTime.IsZero() {
		return false
	}

	// 计算距离上次切换的时间
	elapsed := time.Since(state.LastSwitchTime)
	cooldownDuration := time.Duration(state.SwitchCooldownMin) * time.Minute

	return elapsed < cooldownDuration
}

// GetCooldownRemaining 获取剩余冷却时间（分钟）
func (sm *StateManager) GetCooldownRemaining(domain string) float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[domain]
	if !exists || state.LastSwitchTime.IsZero() {
		return 0
	}

	elapsed := time.Since(state.LastSwitchTime)
	cooldownDuration := time.Duration(state.SwitchCooldownMin) * time.Minute
	remaining := cooldownDuration - elapsed

	if remaining <= 0 {
		return 0
	}

	return remaining.Minutes()
}
