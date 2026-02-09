//go:build linux

package handlers

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"etl/models"
)

// -------------------- Host Metrics (Linux) --------------------
func collectHostMetrics() (*models.HostMetrics, error) {
	metrics := &models.HostMetrics{CPUCores: runtime.NumCPU()}

	cpuUsage, err := readCPUUsagePct()
	if err == nil {
		metrics.CPUUsagePct = cpuUsage
	}

	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err == nil {
		total := uint64(info.Totalram) * uint64(info.Unit)
		free := uint64(info.Freeram) * uint64(info.Unit)
		metrics.MemTotalBytes = total
		metrics.MemUsedBytes = total - free

		swapTotal := uint64(info.Totalswap) * uint64(info.Unit)
		swapFree := uint64(info.Freeswap) * uint64(info.Unit)
		metrics.SwapTotalBytes = swapTotal
		metrics.SwapUsedBytes = swapTotal - swapFree
	}

	var fs syscall.Statfs_t
	if err := syscall.Statfs(".", &fs); err == nil {
		total := fs.Blocks * uint64(fs.Bsize)
		free := fs.Bavail * uint64(fs.Bsize)
		metrics.DiskTotalBytes = total
		metrics.DiskFreeBytes = free
	}

	return metrics, nil
}

func readCPUUsagePct() (float64, error) {
	idle1, total1, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(200 * time.Millisecond)
	idle2, total2, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	if total2 <= total1 {
		return 0, errors.New("cpu total delta invalido")
	}
	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	usage := 100.0 * (1.0 - (idleDelta / totalDelta))
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return usage, nil
}

func readCPUStat() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, 0, errors.New("/proc/stat vazio")
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, errors.New("formato inesperado de /proc/stat")
	}

	var total uint64
	var idle uint64
	for i := 1; i < len(fields); i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		total += val
		if i == 4 || i == 5 { // idle + iowait
			idle += val
		}
	}
	return idle, total, nil
}

