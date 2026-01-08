package api

import (
	"bytes"
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"dnsfailover/internal/monitor"
	"dnsfailover/internal/schedule"
	"dnsfailover/internal/storage"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

//go:embed web
var webFS embed.FS

// Server Web API 服务器
type Server struct {
	cfg             *config.Config
	scheduler       *monitor.Scheduler
	scheduleManager *schedule.Manager
	router          *mux.Router
	server          *http.Server
	mu              sync.RWMutex
}

// NewServer 创建 API 服务器
func NewServer(cfg *config.Config, scheduler *monitor.Scheduler, scheduleManager *schedule.Manager, port int) *Server {
	s := &Server{
		cfg:             cfg,
		scheduler:       scheduler,
		scheduleManager: scheduleManager,
		router:          mux.NewRouter(),
	}

	// 注册路由
	s.registerRoutes()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// registerRoutes 注册路由
func (s *Server) registerRoutes() {
	// 启用 CORS
	s.router.Use(corsMiddleware)

	// 静态文件（前端页面）
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")

	// API 路由
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	api.HandleFunc("/config", s.handleUpdateConfig).Methods("POST")
	api.HandleFunc("/status", s.handleGetStatus).Methods("GET")
	api.HandleFunc("/domains", s.handleGetDomains).Methods("GET")
	api.HandleFunc("/logs", s.handleGetLogs).Methods("GET")
	api.HandleFunc("/logs/clear", s.handleClearLogs).Methods("POST")

	// 定时任务路由
	api.HandleFunc("/schedules", s.handleGetSchedules).Methods("GET")
	api.HandleFunc("/schedules", s.handleCreateSchedule).Methods("POST")
	api.HandleFunc("/schedules/{id}", s.handleGetSchedule).Methods("GET")
	api.HandleFunc("/schedules/{id}", s.handleUpdateSchedule).Methods("PUT")
	api.HandleFunc("/schedules/{id}", s.handleDeleteSchedule).Methods("DELETE")
	api.HandleFunc("/schedules/{id}/run", s.handleRunSchedule).Methods("POST")
	api.HandleFunc("/schedules/{id}/enable", s.handleEnableSchedule).Methods("POST")
	api.HandleFunc("/schedules/{id}/disable", s.handleDisableSchedule).Methods("POST")

	// Webhook 测试路由
	api.HandleFunc("/webhook/test", s.handleTestWebhook).Methods("POST")
}

// Start 启动 API 服务器
func (s *Server) Start() error {
	logger.Infof("[API] Web 管理界面启动: http://localhost%s", s.server.Addr)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("[API] 服务器错误: %v", err)
		}
	}()
	return nil
}

// Stop 停止 API 服务器
func (s *Server) Stop() error {
	logger.Info("[API] 正在停止 Web 服务器...")
	return s.server.Close()
}

// corsMiddleware CORS 中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleIndex 首页
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	content, err := webFS.ReadFile("web/index.html")
	if err != nil {
		logger.Errorf("无法读取内置 index.html: %v", err)
		http.Error(w, "Web 界面加载失败", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// printConfigSummary 打印配置摘要到日志
func (s *Server) printConfigSummary() {
	pingStatus := "禁用"
	if s.cfg.Ping.Enabled {
		pingStatus = fmt.Sprintf("%d 个目标", len(s.cfg.Ping.Domains))
	}

	tcpStatus := "禁用"
	if s.cfg.Tcp.Enabled {
		tcpStatus = fmt.Sprintf("%d 个目标", len(s.cfg.Tcp.Domains))
	}

	httpStatus := "禁用"
	if s.cfg.Http.Enabled {
		httpStatus = fmt.Sprintf("%d 个目标", len(s.cfg.Http.Domains))
	}

	webhookStatus := "未配置"
	if s.cfg.Webhook.URL != "" {
		webhookStatus = s.cfg.Webhook.URL
	}

	logger.Infof("[API] ━━━━━━━━━━ 当前配置 ━━━━━━━━━━")
	logger.Infof("[API] Ping: %s", pingStatus)
	logger.Infof("[API] TCP: %s", tcpStatus)
	logger.Infof("[API] HTTP: %s", httpStatus)
	logger.Infof("[API] Webhook: %s", webhookStatus)
	logger.Infof("[API] 静默期: %d 秒", s.cfg.Webhook.SilencePeriod)
}

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查是否需要打印日志（刷新操作）
	if r.URL.Query().Get("refresh") == "true" {
		s.printConfigSummary()
	}

	response := map[string]interface{}{
		"ping": map[string]interface{}{
			"enabled":            s.cfg.Ping.Enabled,
			"frequency":          s.cfg.Ping.Frequency,
			"failcount":          s.cfg.Ping.FailCount,
			"timeout":            s.cfg.Ping.Timeout,
			"retry":              s.cfg.Ping.Retry,
			"remote_update_freq": s.cfg.Ping.RemoteUpdateFreq,
			"domains":            s.cfg.Ping.Domains,
		},
		"tcp": map[string]interface{}{
			"enabled":   s.cfg.Tcp.Enabled,
			"frequency": s.cfg.Tcp.Frequency,
			"failcount": s.cfg.Tcp.FailCount,
			"timeout":   s.cfg.Tcp.Timeout,
			"retry":     s.cfg.Tcp.Retry,
			"domains":   s.cfg.Tcp.Domains,
		},
		"http": map[string]interface{}{
			"enabled":   s.cfg.Http.Enabled,
			"frequency": s.cfg.Http.Frequency,
			"failcount": s.cfg.Http.FailCount,
			"timeout":   s.cfg.Http.Timeout,
			"retry":     s.cfg.Http.Retry,
			"domains":   s.cfg.Http.Domains,
		},
		"webhook": s.cfg.Webhook,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req storage.FullConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "无效的JSON格式", http.StatusBadRequest)
		return
	}

	// 验证配置 - 至少启用一种探针
	if !req.Ping.Enabled && !req.Tcp.Enabled && !req.Http.Enabled {
		respondError(w, "至少需要启用一种探针 (ping/tcp/http)", http.StatusBadRequest)
		return
	}

	// 设置默认值
	setProbeDefaults(&req.Ping)
	setProbeDefaults(&req.Tcp)
	setProbeDefaults(&req.Http)
	if req.Webhook.Method == "" {
		req.Webhook.Method = "POST"
	}
	if req.Webhook.Timeout == 0 {
		req.Webhook.Timeout = 10
	}
	if req.Webhook.SilencePeriod == 0 {
		req.Webhook.SilencePeriod = 60 // 默认 60 秒静默期
	}

	// 保存到 SQLite
	store := storage.GetStorage()
	if store != nil {
		if err := store.SaveConfig(&req); err != nil {
			respondError(w, fmt.Sprintf("保存配置失败: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// 应用配置到内存
	s.mu.Lock()
	config.ApplyStorageConfig(s.cfg, &req)
	s.mu.Unlock()

	// 更新全局静默期
	if req.Webhook.SilencePeriod > 0 {
		monitor.DefaultSilenceDuration = time.Duration(req.Webhook.SilencePeriod) * time.Second
		logger.Infof("[API] 静默期已更新为 %d 秒", req.Webhook.SilencePeriod)
	}

	logger.Info("[API] 配置已更新并保存到数据库")

	respondSuccess(w, "配置更新成功", nil)
}

// setProbeDefaults 设置探针默认值
func setProbeDefaults(cfg *storage.ProbeConfig) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5
	}
	if cfg.Retry == 0 {
		cfg.Retry = 3
	}
	if cfg.Frequency == 0 {
		cfg.Frequency = 30
	}
	if cfg.FailCount == 0 {
		cfg.FailCount = 3
	}
}

// handleGetStatus 获取运行状态
func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"running":      s.scheduler.IsRunning(),
		"timestamp":    time.Now().Unix(),
		"ping_enabled": s.cfg.Ping.Enabled,
		"ping_count":   len(s.cfg.Ping.Domains),
		"tcp_enabled":  s.cfg.Tcp.Enabled,
		"tcp_count":    len(s.cfg.Tcp.Domains),
		"http_enabled": s.cfg.Http.Enabled,
		"http_count":   len(s.cfg.Http.Domains),
		"webhook_url":  s.cfg.Webhook.URL,
	}

	respondSuccess(w, "获取状态成功", status)
}

// handleGetDomains 获取所有域名状态
func (s *Server) handleGetDomains(w http.ResponseWriter, r *http.Request) {
	// TODO: 从 StateManager 获取域名状态
	respondSuccess(w, "获取域名状态成功", nil)
}

// handleGetLogs 获取内存日志
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// 获取查询参数
	linesStr := r.URL.Query().Get("lines")
	lines := 100 // 默认返回最后100行
	if linesStr != "" {
		fmt.Sscanf(linesStr, "%d", &lines)
	}

	// 从内存缓冲区获取日志
	buffer := logger.GetBuffer()
	if buffer == nil {
		respondError(w, "日志缓冲区未初始化", http.StatusInternalServerError)
		return
	}

	logs := buffer.GetLogs(lines)

	// 格式化日志为字符串
	var content string
	for _, log := range logs {
		content += fmt.Sprintf("[%s] [%s] %s\n",
			log.Timestamp.Format("2006-01-02 15:04:05"),
			log.Level,
			log.Message)
	}

	respondSuccess(w, "获取日志成功", map[string]interface{}{
		"content": content,
		"count":   len(logs),
		"lines":   lines,
	})
}

// handleClearLogs 清空内存日志
func (s *Server) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	buffer := logger.GetBuffer()
	if buffer == nil {
		respondError(w, "日志缓冲区未初始化", http.StatusInternalServerError)
		return
	}

	buffer.Clear()
	logger.Info("[API] 内存日志已清空")
	respondSuccess(w, "日志清理成功", nil)
}

// respondSuccess 成功响应
func respondSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
		"data":    data,
	})
}

// respondError 错误响应
func respondError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
}

// ========== 定时任务 API ==========

// ScheduleTaskRequest 定时任务请求结构
type ScheduleTaskRequest struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Cron        string            `json:"cron"`
	CheckType   string            `json:"check_type"`
	Target      string            `json:"target"`
	Port        int               `json:"port"`
	Timeout     int               `json:"timeout"`
	WebhookURL  string            `json:"webhook_url"`
	WebhookData map[string]string `json:"webhook_data"`
}

// handleGetSchedules 获取所有定时任务
func (s *Server) handleGetSchedules(w http.ResponseWriter, r *http.Request) {
	store := storage.GetStorage()
	if store == nil {
		respondError(w, "数据库未初始化", http.StatusInternalServerError)
		return
	}

	tasks, err := store.GetAllScheduleTasks()
	if err != nil {
		respondError(w, fmt.Sprintf("获取任务列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	respondSuccess(w, "获取成功", tasks)
}

// handleCreateSchedule 创建定时任务
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req ScheduleTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "无效的 JSON 格式", http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if req.Name == "" {
		respondError(w, "任务名称不能为空", http.StatusBadRequest)
		return
	}
	if req.Cron == "" {
		respondError(w, "Cron 表达式不能为空", http.StatusBadRequest)
		return
	}
	if req.Target == "" {
		respondError(w, "检测目标不能为空", http.StatusBadRequest)
		return
	}
	if req.CheckType == "" {
		req.CheckType = "ping"
	}
	if req.Timeout == 0 {
		req.Timeout = 5
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	task := &storage.ScheduleTask{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Enabled:     req.Enabled,
		Cron:        req.Cron,
		CheckType:   req.CheckType,
		Target:      req.Target,
		Port:        req.Port,
		Timeout:     req.Timeout,
		WebhookURL:  req.WebhookURL,
		WebhookData: req.WebhookData,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 保存到数据库
	store := storage.GetStorage()
	if err := store.SaveScheduleTask(task); err != nil {
		respondError(w, fmt.Sprintf("保存任务失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 添加到调度器
	if s.scheduleManager != nil {
		scheduleTask := convertToScheduleTask(task)
		if err := s.scheduleManager.AddTask(scheduleTask); err != nil {
			logger.Warnf("[API] 添加任务到调度器失败: %v", err)
		}
	}

	logger.Infof("[API] 创建定时任务: %s (%s)", task.Name, task.ID)
	respondSuccess(w, "创建成功", task)
}

// handleGetSchedule 获取单个定时任务
func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	store := storage.GetStorage()
	task, err := store.GetScheduleTask(id)
	if err != nil {
		respondError(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	if task == nil {
		respondError(w, "任务不存在", http.StatusNotFound)
		return
	}

	respondSuccess(w, "获取成功", task)
}

// handleUpdateSchedule 更新定时任务
func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req ScheduleTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "无效的 JSON 格式", http.StatusBadRequest)
		return
	}

	store := storage.GetStorage()
	existing, err := store.GetScheduleTask(id)
	if err != nil {
		respondError(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	if existing == nil {
		respondError(w, "任务不存在", http.StatusNotFound)
		return
	}

	// 更新字段
	existing.Name = req.Name
	existing.Enabled = req.Enabled
	existing.Cron = req.Cron
	existing.CheckType = req.CheckType
	existing.Target = req.Target
	existing.Port = req.Port
	existing.Timeout = req.Timeout
	existing.WebhookURL = req.WebhookURL
	existing.WebhookData = req.WebhookData
	existing.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	if err := store.SaveScheduleTask(existing); err != nil {
		respondError(w, fmt.Sprintf("保存失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 更新调度器
	if s.scheduleManager != nil {
		scheduleTask := convertToScheduleTask(existing)
		s.scheduleManager.AddTask(scheduleTask) // AddTask 会自动更新已存在的任务
	}

	logger.Infof("[API] 更新定时任务: %s (%s)", existing.Name, existing.ID)
	respondSuccess(w, "更新成功", existing)
}

// handleDeleteSchedule 删除定时任务
func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	store := storage.GetStorage()
	if err := store.DeleteScheduleTask(id); err != nil {
		respondError(w, fmt.Sprintf("删除失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 从调度器移除
	if s.scheduleManager != nil {
		s.scheduleManager.RemoveTask(id)
	}

	logger.Infof("[API] 删除定时任务: %s", id)
	respondSuccess(w, "删除成功", nil)
}

// handleRunSchedule 立即执行任务
func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if s.scheduleManager == nil {
		respondError(w, "调度器未初始化", http.StatusInternalServerError)
		return
	}

	result, err := s.scheduleManager.RunTaskNow(id)
	if err != nil {
		respondError(w, fmt.Sprintf("执行失败: %v", err), http.StatusInternalServerError)
		return
	}

	respondSuccess(w, "执行完成", result)
}

// handleEnableSchedule 启用任务
func (s *Server) handleEnableSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	store := storage.GetStorage()
	task, err := store.GetScheduleTask(id)
	if err != nil || task == nil {
		respondError(w, "任务不存在", http.StatusNotFound)
		return
	}

	task.Enabled = true
	task.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	store.SaveScheduleTask(task)

	if s.scheduleManager != nil {
		s.scheduleManager.EnableTask(id)
	}

	respondSuccess(w, "已启用", nil)
}

// handleDisableSchedule 禁用任务
func (s *Server) handleDisableSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	store := storage.GetStorage()
	task, err := store.GetScheduleTask(id)
	if err != nil || task == nil {
		respondError(w, "任务不存在", http.StatusNotFound)
		return
	}

	task.Enabled = false
	task.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	store.SaveScheduleTask(task)

	if s.scheduleManager != nil {
		s.scheduleManager.DisableTask(id)
	}

	respondSuccess(w, "已禁用", nil)
}

// convertToScheduleTask 转换存储任务到调度任务
func convertToScheduleTask(st *storage.ScheduleTask) *schedule.Task {
	task := &schedule.Task{
		ID:          st.ID,
		Name:        st.Name,
		Enabled:     st.Enabled,
		Cron:        st.Cron,
		CheckType:   schedule.CheckType(st.CheckType),
		Target:      st.Target,
		Port:        st.Port,
		Timeout:     st.Timeout,
		WebhookURL:  st.WebhookURL,
		WebhookData: st.WebhookData,
	}

	if st.CreatedAt != "" {
		task.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", st.CreatedAt)
	}
	if st.UpdatedAt != "" {
		task.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", st.UpdatedAt)
	}
	if st.LastRunAt != nil {
		t, _ := time.Parse("2006-01-02 15:04:05", *st.LastRunAt)
		task.LastRunAt = &t
	}
	task.LastResult = st.LastResult

	return task
}

// handleTestWebhook 测试 Webhook 发送
func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Timeout int               `json:"timeout"`
		Headers map[string]string `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "无效的请求: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondError(w, "Webhook URL 不能为空", http.StatusBadRequest)
		return
	}

	// 构建测试数据
	testData := map[string]interface{}{
		"type":       "test",
		"probe_type": "test",
		"target":     "test.example.com",
		"fail_count": 0,
		"threshold":  3,
		"error":      "",
		"timestamp":  time.Now().Unix(),
		"message":    "这是一条 Webhook 测试消息",
	}

	body, _ := json.Marshal(testData)

	// 创建请求
	method := req.Method
	if method == "" {
		method = "POST"
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	httpReq, err := http.NewRequest(method, req.URL, bytes.NewReader(body))
	if err != nil {
		respondError(w, "创建请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		respondError(w, "发送失败: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respondError(w, fmt.Sprintf("目标返回错误: %d %s", resp.StatusCode, resp.Status), http.StatusBadGateway)
		return
	}

	respondSuccess(w, fmt.Sprintf("发送成功，目标返回: %d %s", resp.StatusCode, resp.Status), nil)
}
