package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// Storage SQLite 存储管理器
type Storage struct {
	db *sql.DB
	mu sync.RWMutex
}

// ProbeConfig 探针配置（用于 JSON 序列化）
type ProbeConfig struct {
	Enabled          bool     `json:"enabled"`
	Frequency        int      `json:"frequency"`
	FailCount        int      `json:"failcount"`
	Timeout          int      `json:"timeout"`
	Retry            int      `json:"retry"`
	RemoteUpdateFreq int      `json:"remote_update_freq"`
	Domains          []string `json:"domains"`
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Timeout       int               `json:"timeout"`
	Retry         int               `json:"retry"`
	SilencePeriod int               `json:"silence_period"` // 静默期（秒）
}

// FullConfig 完整配置结构
type FullConfig struct {
	Ping    ProbeConfig   `json:"ping"`
	Tcp     ProbeConfig   `json:"tcp"`
	Http    ProbeConfig   `json:"http"`
	Webhook WebhookConfig `json:"webhook"`
}

var (
	instance *Storage
	once     sync.Once
)

// GetStorage 获取存储单例
func GetStorage() *Storage {
	return instance
}

// Init 初始化 SQLite 存储
func Init(dbPath string) (*Storage, error) {
	var err error
	once.Do(func() {
		var db *sql.DB
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return
		}

		// 测试连接
		if err = db.Ping(); err != nil {
			return
		}

		instance = &Storage{db: db}

		// 创建配置表
		err = instance.createTables()
	})

	if err != nil {
		return nil, fmt.Errorf("初始化 SQLite 失败: %w", err)
	}

	return instance, nil
}

// createTables 创建数据库表
func (s *Storage) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS schedule_tasks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		cron TEXT NOT NULL,
		check_type TEXT NOT NULL,
		target TEXT NOT NULL,
		port INTEGER DEFAULT 0,
		timeout INTEGER DEFAULT 5,
		webhook_url TEXT,
		webhook_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_run_at DATETIME,
		last_result TEXT
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close 关闭数据库连接
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveConfig 保存完整配置
func (s *Storage) SaveConfig(cfg *FullConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO config (key, value, updated_at) 
		VALUES ('main_config', ?, CURRENT_TIMESTAMP)
	`, string(data))

	if err != nil {
		return fmt.Errorf("保存配置到 SQLite 失败: %w", err)
	}

	return nil
}

// LoadConfig 加载完整配置
func (s *Storage) LoadConfig() (*FullConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = 'main_config'`).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil // 没有配置，返回 nil
	}
	if err != nil {
		return nil, fmt.Errorf("从 SQLite 读取配置失败: %w", err)
	}

	var cfg FullConfig
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return nil, fmt.Errorf("解析配置 JSON 失败: %w", err)
	}

	return &cfg, nil
}

// HasConfig 检查是否有配置
func (s *Storage) HasConfig() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM config WHERE key = 'main_config'`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *FullConfig {
	return &FullConfig{
		Ping: ProbeConfig{
			Enabled:          true,
			Frequency:        30,
			FailCount:        3,
			Timeout:          5,
			Retry:            3,
			RemoteUpdateFreq: 60,
			Domains:          []string{},
		},
		Tcp: ProbeConfig{
			Enabled:          false,
			Frequency:        30,
			FailCount:        3,
			Timeout:          5,
			Retry:            3,
			RemoteUpdateFreq: 60,
			Domains:          []string{},
		},
		Http: ProbeConfig{
			Enabled:          false,
			Frequency:        30,
			FailCount:        3,
			Timeout:          10,
			Retry:            3,
			RemoteUpdateFreq: 60,
			Domains:          []string{},
		},
		Webhook: WebhookConfig{
			URL:     "",
			Method:  "POST",
			Timeout: 10,
			Headers: make(map[string]string),
		},
	}
}

// ScheduleTask 定时任务结构（存储用）
type ScheduleTask struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Cron        string            `json:"cron"`
	CheckType   string            `json:"check_type"`
	Target      string            `json:"target"`
	Port        int               `json:"port"`
	Timeout     int               `json:"timeout"`
	WebhookURL  string            `json:"webhook_url"`
	WebhookData map[string]string `json:"webhook_data"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	LastRunAt   *string           `json:"last_run_at"`
	LastResult  string            `json:"last_result"`
}

// SaveScheduleTask 保存定时任务
func (s *Storage) SaveScheduleTask(task *ScheduleTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhookData := ""
	if task.WebhookData != nil {
		data, _ := json.Marshal(task.WebhookData)
		webhookData = string(data)
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO schedule_tasks 
		(id, name, enabled, cron, check_type, target, port, timeout, webhook_url, webhook_data, created_at, updated_at, last_run_at, last_result)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.Name, task.Enabled, task.Cron, task.CheckType, task.Target, task.Port, task.Timeout,
		task.WebhookURL, webhookData, task.CreatedAt, task.UpdatedAt, task.LastRunAt, task.LastResult)

	if err != nil {
		return fmt.Errorf("保存定时任务失败: %w", err)
	}

	return nil
}

// GetScheduleTask 获取单个定时任务
func (s *Storage) GetScheduleTask(id string) (*ScheduleTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var task ScheduleTask
	var enabled int
	var webhookData string
	var lastRunAt sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, enabled, cron, check_type, target, port, timeout, webhook_url, webhook_data, created_at, updated_at, last_run_at, last_result
		FROM schedule_tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.Name, &enabled, &task.Cron, &task.CheckType, &task.Target, &task.Port, &task.Timeout,
		&task.WebhookURL, &webhookData, &task.CreatedAt, &task.UpdatedAt, &lastRunAt, &task.LastResult)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询定时任务失败: %w", err)
	}

	task.Enabled = enabled == 1
	if webhookData != "" {
		json.Unmarshal([]byte(webhookData), &task.WebhookData)
	}
	if lastRunAt.Valid {
		task.LastRunAt = &lastRunAt.String
	}

	return &task, nil
}

// GetAllScheduleTasks 获取所有定时任务
func (s *Storage) GetAllScheduleTasks() ([]*ScheduleTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, enabled, cron, check_type, target, port, timeout, webhook_url, webhook_data, created_at, updated_at, last_run_at, last_result
		FROM schedule_tasks ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询定时任务列表失败: %w", err)
	}
	defer rows.Close()

	var tasks []*ScheduleTask
	for rows.Next() {
		var task ScheduleTask
		var enabled int
		var webhookData string
		var lastRunAt sql.NullString

		err := rows.Scan(&task.ID, &task.Name, &enabled, &task.Cron, &task.CheckType, &task.Target, &task.Port, &task.Timeout,
			&task.WebhookURL, &webhookData, &task.CreatedAt, &task.UpdatedAt, &lastRunAt, &task.LastResult)
		if err != nil {
			return nil, fmt.Errorf("读取定时任务失败: %w", err)
		}

		task.Enabled = enabled == 1
		if webhookData != "" {
			json.Unmarshal([]byte(webhookData), &task.WebhookData)
		}
		if lastRunAt.Valid {
			task.LastRunAt = &lastRunAt.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// DeleteScheduleTask 删除定时任务
func (s *Storage) DeleteScheduleTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`DELETE FROM schedule_tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除定时任务失败: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("任务不存在: %s", id)
	}

	return nil
}

// UpdateScheduleTaskStatus 更新任务执行状态
func (s *Storage) UpdateScheduleTaskStatus(id string, lastRunAt string, lastResult string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE schedule_tasks SET last_run_at = ?, last_result = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, lastRunAt, lastResult, id)

	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	return nil
}
