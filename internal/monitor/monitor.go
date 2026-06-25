package monitor

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cedar2025/xboard-node/internal/nlog"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

var startTime = time.Now()

func init() {
	// Warm up the CPU sampler. The first cpu.Percent call with interval=0
	// always returns 0% because it has no prior sample. This throwaway call
	// seeds the baseline so subsequent Collect() calls return real values.
	cpu.Percent(500*time.Millisecond, false)

	// Seed the network baseline so the first Collect() can compute rates.
	collectNetSpeed()
}

// Status holds system resource metrics
type Status struct {
	Uptime     uint64
	CPU        float64
	CPUPerCore []float64
	Load1      float64
	Load5      float64
	Load15     float64
	MemTotal   uint64
	MemUsed    uint64
	SwapTotal  uint64
	SwapUsed   uint64
	DiskTotal  uint64
	DiskUsed   uint64
	Goroutines int

	// Network speed (bytes/sec), -1 means unavailable (first sample).
	NetInSpeed  float64
	NetOutSpeed float64

	// GC metrics (process-wide)
	NumGC       uint32
	LastPauseMS float64
}

// netBaseline tracks the previous network counters for rate calculation.
var (
	netMu       sync.Mutex
	netPrevRecv uint64
	netPrevSent uint64
	netPrevTime time.Time
	netHasBase  bool
)

// skipInterface returns true for loopback and common virtual interfaces.
func skipInterface(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"lo", "docker", "veth", "br-", "virbr", "vnet", "tun", "tap"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// collectNetSpeed calculates network in/out bytes per second since last call.
// Returns -1, -1 on first call or if counters decreased (reboot).
func collectNetSpeed() (inSpeed, outSpeed float64) {
	counters, err := net.IOCounters(true) // per-interface
	if err != nil {
		nlog.Core().Debug("failed to get network counters", "error", err)
		return -1, -1
	}

	var totalRecv, totalSent uint64
	for _, c := range counters {
		if skipInterface(c.Name) {
			continue
		}
		totalRecv += c.BytesRecv
		totalSent += c.BytesSent
	}

	now := time.Now()

	netMu.Lock()
	defer netMu.Unlock()

	if !netHasBase {
		netPrevRecv, netPrevSent, netPrevTime, netHasBase = totalRecv, totalSent, now, true
		return -1, -1
	}

	elapsed := now.Sub(netPrevTime).Seconds()
	if elapsed <= 0 {
		return -1, -1
	}

	// Counter decreased → system reboot or interface reset; reset baseline.
	if totalRecv < netPrevRecv || totalSent < netPrevSent {
		netPrevRecv, netPrevSent, netPrevTime = totalRecv, totalSent, now
		return -1, -1
	}

	inSpeed = float64(totalRecv-netPrevRecv) / elapsed
	outSpeed = float64(totalSent-netPrevSent) / elapsed

	netPrevRecv, netPrevSent, netPrevTime = totalRecv, totalSent, now
	return inSpeed, outSpeed
}

// Collect gathers current system metrics
func Collect() Status {
	var s Status

	s.Uptime = uint64(time.Since(startTime).Seconds())

	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		s.CPU = cpuPercent[0]
	} else if err != nil {
		nlog.Core().Debug("failed to get CPU usage", "error", err)
	}

	// Per-core CPU usage (best-effort; safe if it fails).
	if perCore, err := cpu.Percent(0, true); err == nil && len(perCore) > 0 {
		s.CPUPerCore = perCore
	}

	if loadAvg, err := load.Avg(); err == nil {
		s.Load1 = loadAvg.Load1
		s.Load5 = loadAvg.Load5
		s.Load15 = loadAvg.Load15
	}

	if vmStat, err := mem.VirtualMemory(); err == nil {
		s.MemTotal = vmStat.Total
		s.MemUsed = vmStat.Used
	}

	if swapStat, err := mem.SwapMemory(); err == nil {
		s.SwapTotal = swapStat.Total
		s.SwapUsed = swapStat.Used
	}

	if diskStat, err := disk.Usage("/"); err == nil {
		s.DiskTotal = diskStat.Total
		s.DiskUsed = diskStat.Used
	}

	s.NetInSpeed, s.NetOutSpeed = collectNetSpeed()

	// GC metrics
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	s.Goroutines = runtime.NumGoroutine()
	s.NumGC = ms.NumGC
	if ms.NumGC > 0 {
		// PauseNs is a ring buffer of the most recent GC pause times.
		idx := (ms.NumGC - 1) % uint32(len(ms.PauseNs))
		s.LastPauseMS = float64(ms.PauseNs[idx]) / 1e6
	}

	return s
}
