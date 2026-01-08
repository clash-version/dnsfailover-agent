package monitor

import (
	"sync"
	"time"
)

// DefaultSilenceDuration 默认静默期时间（发送告警后暂停检测的时间）
// 可通过配置覆盖
var DefaultSilenceDuration = 60 * time.Second

// DomainState 域名运行时状态（仅存在于内存中）
type DomainState struct {
	Domain        string    // 域名/目标
	FailCount     int       // 当前周期内的连续失败次数
	LastAlertTime time.Time // 最后一次告警时间
	IsDown        bool      // 当前是否处于故障状态
	SilenceUntil  time.Time // 静默期截止时间（此时间前不进行检测）
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

// InitDomain 初始化域名状态
func (sm *StateManager) InitDomain(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.states[domain] = &DomainState{
		Domain:    domain,
		FailCount: 0,
		IsDown:    false,
	}
}

// GetState 获取域名状态
func (sm *StateManager) GetState(domain string) *DomainState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[domain]; exists {
		return state
	}
	return &DomainState{Domain: domain}
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
		Domain:    domain,
		FailCount: 1,
		IsDown:    false,
	}
	return 1
}

// ResetFailCount 重置失败计数为0
func (sm *StateManager) ResetFailCount(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.FailCount = 0
		state.IsDown = false
	}
}

// MarkDown 标记为故障状态，并设置静默期
func (sm *StateManager) MarkDown(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.IsDown = true
		state.LastAlertTime = time.Now()
		state.SilenceUntil = time.Now().Add(DefaultSilenceDuration)
	}
}

// MarkDownWithSilence 标记为故障状态，指定静默期
func (sm *StateManager) MarkDownWithSilence(domain string, silenceDuration time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.IsDown = true
		state.LastAlertTime = time.Now()
		state.SilenceUntil = time.Now().Add(silenceDuration)
	}
}

// IsSilenced 检查目标是否在静默期内
func (sm *StateManager) IsSilenced(domain string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[domain]; exists {
		return time.Now().Before(state.SilenceUntil)
	}
	return false
}

// GetSilenceRemaining 获取剩余静默时间
func (sm *StateManager) GetSilenceRemaining(domain string) time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[domain]; exists {
		remaining := time.Until(state.SilenceUntil)
		if remaining > 0 {
			return remaining
		}
	}
	return 0
}

// ClearSilence 清除静默期
func (sm *StateManager) ClearSilence(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, exists := sm.states[domain]; exists {
		state.SilenceUntil = time.Time{}
	}
}

// RemoveDomain 移除域名状态
func (sm *StateManager) RemoveDomain(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, domain)
}

// GetAllStates 获取所有域名状态
func (sm *StateManager) GetAllStates() map[string]*DomainState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*DomainState)
	for k, v := range sm.states {
		result[k] = &DomainState{
			Domain:        v.Domain,
			FailCount:     v.FailCount,
			LastAlertTime: v.LastAlertTime,
			IsDown:        v.IsDown,
		}
	}
	return result
}
