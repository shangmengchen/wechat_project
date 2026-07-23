package adminview

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"couple-mini/backend/internal/model"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	mu           sync.RWMutex
	history      []model.AdminSystemPoint
	historyCap   = 120
	startAt      = time.Now()
	tickerStop   chan struct{}
	started      bool
	appName      string
	appVersion   string
	runMode      string
	requestTotal atomic.Uint64
	errorTotal   atomic.Uint64
)

func Start(name, version, mode string, interval time.Duration, limit int) {
	mu.Lock()
	defer mu.Unlock()

	appName = name
	appVersion = version
	runMode = mode
	startAt = time.Now()
	if limit > 0 {
		historyCap = limit
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if started {
		return
	}
	started = true
	tickerStop = make(chan struct{})
	recordSnapshotLocked()
	go run(interval)
}

func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if !started {
		return
	}
	close(tickerStop)
	started = false
}

func ObserveRequest(status int) {
	requestTotal.Add(1)
	if status >= 400 {
		errorTotal.Add(1)
	}
}

func SnapshotHistory() []model.AdminSystemPoint {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]model.AdminSystemPoint, len(history))
	copy(out, history)
	return out
}

func RuntimeSummary() model.AdminRuntime {
	point := latestPoint()
	return model.AdminRuntime{
		AppName:           appName,
		Version:           appVersion,
		RunMode:           runMode,
		StartAt:           startAt,
		UptimeSeconds:     int64(time.Since(startAt).Seconds()),
		Goroutines:        runtime.NumGoroutine(),
		RequestTotal:      requestTotal.Load(),
		ErrorTotal:        errorTotal.Load(),
		LastCPUPercent:    point.CPUPercent,
		LastMemoryPercent: point.MemoryPercent,
		ProcessMemoryMB:   point.ProcessMemoryMB,
	}
}

func RequestTotal() uint64 {
	return requestTotal.Load()
}

func ErrorTotal() uint64 {
	return errorTotal.Load()
}

func run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mu.Lock()
			recordSnapshotLocked()
			mu.Unlock()
		case <-tickerStop:
			return
		}
	}
}

func recordSnapshotLocked() {
	history = append(history, capturePoint())
	if len(history) > historyCap {
		history = history[len(history)-historyCap:]
	}
}

func capturePoint() model.AdminSystemPoint {
	cpuPercent := 0.0
	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		cpuPercent = values[0]
	}

	memoryPercent := 0.0
	if vm, err := mem.VirtualMemory(); err == nil {
		memoryPercent = vm.UsedPercent
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return model.AdminSystemPoint{
		Timestamp:       time.Now(),
		CPUPercent:      cpuPercent,
		MemoryPercent:   memoryPercent,
		ProcessMemoryMB: float64(memStats.Alloc) / 1024.0 / 1024.0,
		Goroutines:      runtime.NumGoroutine(),
		RequestTotal:    requestTotal.Load(),
		ErrorTotal:      errorTotal.Load(),
	}
}

func latestPoint() model.AdminSystemPoint {
	mu.RLock()
	defer mu.RUnlock()
	if len(history) == 0 {
		return capturePoint()
	}
	return history[len(history)-1]
}

func Hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}
