package logger

import (
	"container/ring"
	"sync"
	"time"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// LogBuffer 内存日志缓冲区
type LogBuffer struct {
	buffer *ring.Ring
	mu     sync.RWMutex
	size   int
}

var globalBuffer *LogBuffer

// InitBuffer 初始化日志缓冲区
func InitBuffer(size int) {
	globalBuffer = &LogBuffer{
		buffer: ring.New(size),
		size:   size,
	}
}

// AddLog 添加日志到缓冲区
func (lb *LogBuffer) AddLog(level, message string) {
	if lb == nil {
		return
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	lb.buffer.Value = entry
	lb.buffer = lb.buffer.Next()
}

// GetLogs 获取最近的N条日志
func (lb *LogBuffer) GetLogs(n int) []LogEntry {
	if lb == nil {
		return []LogEntry{}
	}

	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var logs []LogEntry
	count := 0

	lb.buffer.Do(func(v interface{}) {
		if v != nil && count < n {
			logs = append(logs, v.(LogEntry))
			count++
		}
	})

	return logs
}

// Clear 清空日志缓冲区
func (lb *LogBuffer) Clear() {
	if lb == nil {
		return
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.buffer = ring.New(lb.size)
}

// GetBuffer 获取全局缓冲区
func GetBuffer() *LogBuffer {
	return globalBuffer
}
