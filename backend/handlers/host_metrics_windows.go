//go:build windows

package handlers

import (
	"errors"
	"runtime"
	"sort"

	"etl/models"

	"github.com/StackExchange/wmi"
)

// -------------------- Host Metrics (Windows) --------------------
func collectHostMetrics() (*models.HostMetrics, error) {
	metrics := &models.HostMetrics{CPUCores: runtime.NumCPU()}

	if cpuUsage, err := readCPUUsagePct(); err == nil {
		metrics.CPUUsagePct = cpuUsage
	}
	if memTotal, memUsed, err := readMemoryBytes(); err == nil {
		metrics.MemTotalBytes = memTotal
		metrics.MemUsedBytes = memUsed
	}
	if swapTotal, swapUsed, err := readSwapBytes(); err == nil {
		metrics.SwapTotalBytes = swapTotal
		metrics.SwapUsedBytes = swapUsed
	}
	if diskTotal, diskFree, err := readDiskBytes(); err == nil {
		metrics.DiskTotalBytes = diskTotal
		metrics.DiskFreeBytes = diskFree
	}

	return metrics, nil
}

type winCPUPerf struct {
	Name                 string
	PercentProcessorTime uint64
}

func readCPUUsagePct() (float64, error) {
	var rows []winCPUPerf
	if err := wmi.Query("SELECT Name, PercentProcessorTime FROM Win32_PerfFormattedData_PerfOS_Processor", &rows); err != nil {
		return 0, err
	}
	for _, r := range rows {
		if r.Name == "_Total" {
			if r.PercentProcessorTime > 100 {
				return 100, nil
			}
			return float64(r.PercentProcessorTime), nil
		}
	}
	return 0, errors.New("cpu total nao encontrado")
}

type winOS struct {
	TotalVisibleMemorySize uint64
	FreePhysicalMemory     uint64
}

func readMemoryBytes() (uint64, uint64, error) {
	var rows []winOS
	if err := wmi.Query("SELECT TotalVisibleMemorySize, FreePhysicalMemory FROM Win32_OperatingSystem", &rows); err != nil {
		return 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, errors.New("informacoes de memoria nao encontradas")
	}
	total := rows[0].TotalVisibleMemorySize * 1024
	free := rows[0].FreePhysicalMemory * 1024
	if total < free {
		return total, 0, nil
	}
	return total, total - free, nil
}

type winPageFile struct {
	AllocatedBaseSize uint64
	CurrentUsage      uint64
}

func readSwapBytes() (uint64, uint64, error) {
	var rows []winPageFile
	if err := wmi.Query("SELECT AllocatedBaseSize, CurrentUsage FROM Win32_PageFileUsage", &rows); err != nil {
		return 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, errors.New("pagefile nao encontrado")
	}
	var totalMB uint64
	var usedMB uint64
	for _, r := range rows {
		totalMB += r.AllocatedBaseSize
		usedMB += r.CurrentUsage
	}
	return totalMB * 1024 * 1024, usedMB * 1024 * 1024, nil
}

type winLogicalDisk struct {
	DeviceID  string
	DriveType uint32
	FreeSpace uint64
	Size      uint64
}

func readDiskBytes() (uint64, uint64, error) {
	var rows []winLogicalDisk
	if err := wmi.Query("SELECT DeviceID, DriveType, FreeSpace, Size FROM Win32_LogicalDisk WHERE DriveType = 3", &rows); err != nil {
		return 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, errors.New("discos locais nao encontrados")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].DeviceID < rows[j].DeviceID })
	var total uint64
	var free uint64
	for _, d := range rows {
		total += d.Size
		free += d.FreeSpace
	}
	return total, free, nil
}
