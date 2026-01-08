package schedule

import (
	"bytes"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/probe"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Manager 定时任务管理器
type Manager struct {
	cron      *cron.Cron
	tasks     map[string]*Task        // 任务列表
	cronIDs   map[string]cron.EntryID // 任务ID -> cron EntryID 映射
	mu        sync.RWMutex
	isRunning bool

	// 检测器
	pingChecker *probe.PingChecker
	tcpChecker  *probe.TCPChecker
	httpChecker *probe.HTTPChecker

	// 任务变更回调（用于持久化）
	onTaskUpdate func(task *Task) error
}

// NewManager 创建定时任务管理器
func NewManager() *Manager {
	return &Manager{
		cron:        cron.New(cron.WithSeconds()), // 支持秒级精度
		tasks:       make(map[string]*Task),
		cronIDs:     make(map[string]cron.EntryID),
		pingChecker: probe.NewPingChecker(),
		tcpChecker:  probe.NewTCPChecker(),
		httpChecker: probe.NewHTTPChecker(10 * time.Second),
	}
}

// SetTaskUpdateCallback 设置任务更新回调
func (m *Manager) SetTaskUpdateCallback(callback func(task *Task) error) {
	m.onTaskUpdate = callback
}

// Start 启动调度器
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return
	}

	m.cron.Start()
	m.isRunning = true
	logger.Info("[Schedule] 定时任务调度器已启动")
}

// Stop 停止调度器
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	m.cron.Stop()
	m.isRunning = false
	logger.Info("[Schedule] 定时任务调度器已停止")
}

// AddTask 添加任务
func (m *Manager) AddTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证 cron 表达式
	if _, err := cron.ParseStandard(task.Cron); err != nil {
		return fmt.Errorf("无效的 cron 表达式: %w", err)
	}

	// 如果任务已存在，先移除
	if oldID, exists := m.cronIDs[task.ID]; exists {
		m.cron.Remove(oldID)
	}

	// 添加到 cron（使用标准5位格式，cron库会自动处理）
	if task.Enabled {
		entryID, err := m.cron.AddFunc(task.Cron, func() {
			m.executeTask(task.ID)
		})
		if err != nil {
			return fmt.Errorf("添加 cron 任务失败: %w", err)
		}
		m.cronIDs[task.ID] = entryID
	}

	m.tasks[task.ID] = task
	logger.Infof("[Schedule] 任务已添加: %s (%s) - %s", task.Name, task.ID, task.Cron)

	return nil
}

// RemoveTask 移除任务
func (m *Manager) RemoveTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entryID, exists := m.cronIDs[taskID]; exists {
		m.cron.Remove(entryID)
		delete(m.cronIDs, taskID)
	}

	if _, exists := m.tasks[taskID]; !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	delete(m.tasks, taskID)
	logger.Infof("[Schedule] 任务已移除: %s", taskID)

	return nil
}

// GetTask 获取任务
func (m *Manager) GetTask(taskID string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	return task, exists
}

// GetAllTasks 获取所有任务
func (m *Manager) GetAllTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// EnableTask 启用任务
func (m *Manager) EnableTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	if task.Enabled {
		return nil // 已启用
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()

	// 添加到 cron
	entryID, err := m.cron.AddFunc(task.Cron, func() {
		m.executeTask(taskID)
	})
	if err != nil {
		task.Enabled = false
		return fmt.Errorf("启用任务失败: %w", err)
	}
	m.cronIDs[taskID] = entryID

	logger.Infof("[Schedule] 任务已启用: %s", task.Name)
	return nil
}

// DisableTask 禁用任务
func (m *Manager) DisableTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	if !task.Enabled {
		return nil // 已禁用
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	// 从 cron 移除
	if entryID, exists := m.cronIDs[taskID]; exists {
		m.cron.Remove(entryID)
		delete(m.cronIDs, taskID)
	}

	logger.Infof("[Schedule] 任务已禁用: %s", task.Name)
	return nil
}

// executeTask 执行任务
func (m *Manager) executeTask(taskID string) {
	m.mu.RLock()
	task, exists := m.tasks[taskID]
	m.mu.RUnlock()

	if !exists || !task.Enabled {
		return
	}

	logger.Infof("[Schedule] 开始执行任务: %s (%s)", task.Name, taskID)

	// 执行检测
	available, message := m.checkTarget(task)

	now := time.Now()
	result := &TaskResult{
		TaskID:     taskID,
		TaskName:   task.Name,
		Success:    available,
		CheckType:  task.CheckType,
		Target:     task.Target,
		Message:    message,
		ExecutedAt: now,
	}

	// 更新任务状态
	m.mu.Lock()
	task.LastRunAt = &now
	if available {
		task.LastResult = "可用"
	} else {
		task.LastResult = "不可用: " + message
	}
	m.mu.Unlock()

	// 发送 Webhook 通知
	if task.WebhookURL != "" {
		webhookSent := m.sendWebhook(task, available, message, now)
		result.WebhookSent = webhookSent
	}

	// 回调更新（持久化）
	if m.onTaskUpdate != nil {
		if err := m.onTaskUpdate(task); err != nil {
			logger.Errorf("[Schedule] 更新任务状态失败: %v", err)
		}
	}

	if available {
		logger.Infof("[Schedule] 任务执行完成: %s - 目标可用", task.Name)
	} else {
		logger.Warnf("[Schedule] 任务执行完成: %s - 目标不可用: %s", task.Name, message)
	}
}

// checkTarget 检测目标
func (m *Manager) checkTarget(task *Task) (bool, string) {
	timeout := time.Duration(task.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	switch task.CheckType {
	case CheckTypePing:
		result := m.pingChecker.Check(task.Target, timeout)
		if result.Success {
			return true, fmt.Sprintf("Ping 成功, 延迟: %v", result.Latency)
		}
		return false, result.Error.Error()

	case CheckTypeTCP:
		target := task.Target
		if task.Port > 0 {
			target = fmt.Sprintf("%s:%d", task.Target, task.Port)
		}
		result := m.tcpChecker.Check(target, timeout)
		if result.Success {
			return true, fmt.Sprintf("TCP 连接成功, 耗时: %v", result.Latency)
		}
		return false, result.Error.Error()

	case CheckTypeHTTP:
		result := m.httpChecker.Check(task.Target, timeout)
		if result.Success {
			return true, fmt.Sprintf("HTTP 检测成功, 耗时: %v", result.Latency)
		}
		return false, result.Error.Error()

	default:
		return false, "未知的检测类型"
	}
}

// sendWebhook 发送 Webhook 通知
func (m *Manager) sendWebhook(task *Task, available bool, message string, executedAt time.Time) bool {
	payload := WebhookPayload{
		Event:      "schedule_check",
		TaskID:     task.ID,
		TaskName:   task.Name,
		CheckType:  task.CheckType,
		Target:     task.Target,
		Available:  available,
		Message:    message,
		CustomData: task.WebhookData,
		ExecutedAt: executedAt,
		Timestamp:  executedAt.Unix(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("[Schedule] 序列化 Webhook 数据失败: %v", err)
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(task.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Errorf("[Schedule] 发送 Webhook 失败: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.Infof("[Schedule] Webhook 发送成功: %s -> %s", task.Name, task.WebhookURL)
		return true
	}

	logger.Warnf("[Schedule] Webhook 返回非成功状态码: %d", resp.StatusCode)
	return false
}

// RunTaskNow 立即执行任务（手动触发）
func (m *Manager) RunTaskNow(taskID string) (*TaskResult, error) {
	m.mu.RLock()
	task, exists := m.tasks[taskID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	logger.Infof("[Schedule] 手动执行任务: %s", task.Name)

	// 执行检测
	available, message := m.checkTarget(task)

	now := time.Now()
	result := &TaskResult{
		TaskID:     taskID,
		TaskName:   task.Name,
		Success:    available,
		CheckType:  task.CheckType,
		Target:     task.Target,
		Message:    message,
		ExecutedAt: now,
	}

	// 更新任务状态
	m.mu.Lock()
	task.LastRunAt = &now
	if available {
		task.LastResult = "可用 (手动执行)"
	} else {
		task.LastResult = "不可用 (手动执行): " + message
	}
	m.mu.Unlock()

	// 发送 Webhook
	if task.WebhookURL != "" {
		result.WebhookSent = m.sendWebhook(task, available, message, now)
	}

	// 回调更新
	if m.onTaskUpdate != nil {
		m.onTaskUpdate(task)
	}

	return result, nil
}

// IsRunning 检查是否在运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetTaskCount 获取任务数量
func (m *Manager) GetTaskCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tasks)
}
