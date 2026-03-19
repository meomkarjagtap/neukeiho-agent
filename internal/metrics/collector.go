package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Snapshot holds a single point-in-time reading of all metrics.
type Snapshot struct {
	Timestamp time.Time
	CPU       float64 // percent
	Memory    float64 // percent
	Disk      float64 // percent (root filesystem)
	NetworkRx float64 // Mbps
	NetworkTx float64 // Mbps
}

// Collector reads metrics from /proc.
type Collector struct {
	prevCPU     cpuStat
	prevNetwork networkStat
	prevTime    time.Time
}

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq uint64
}

type networkStat struct {
	rx, tx uint64
}

// NewCollector creates a new metrics Collector.
func NewCollector() *Collector {
	c := &Collector{}
	c.prevCPU, _ = readCPUStat()
	c.prevNetwork, _ = readNetworkStat()
	c.prevTime = time.Now()
	return c
}

// Collect reads current metrics and returns a Snapshot.
func (c *Collector) Collect() (Snapshot, error) {
	now := time.Now()
	elapsed := now.Sub(c.prevTime).Seconds()

	cpu, err := readCPUStat()
	if err != nil {
		return Snapshot{}, fmt.Errorf("cpu: %w", err)
	}

	mem, err := readMemoryPercent()
	if err != nil {
		return Snapshot{}, fmt.Errorf("memory: %w", err)
	}

	disk, err := readDiskPercent("/")
	if err != nil {
		return Snapshot{}, fmt.Errorf("disk: %w", err)
	}

	net, err := readNetworkStat()
	if err != nil {
		return Snapshot{}, fmt.Errorf("network: %w", err)
	}

	cpuPercent := calcCPUPercent(c.prevCPU, cpu)

	rxMbps := float64(net.rx-c.prevNetwork.rx) * 8 / 1e6 / elapsed
	txMbps := float64(net.tx-c.prevNetwork.tx) * 8 / 1e6 / elapsed

	c.prevCPU = cpu
	c.prevNetwork = net
	c.prevTime = now

	return Snapshot{
		Timestamp: now,
		CPU:       cpuPercent,
		Memory:    mem,
		Disk:      disk,
		NetworkRx: rxMbps,
		NetworkTx: txMbps,
	}, nil
}

func readCPUStat() (cpuStat, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			break
		}
		nums := make([]uint64, 7)
		for i := 0; i < 7; i++ {
			nums[i], _ = strconv.ParseUint(fields[i+1], 10, 64)
		}
		return cpuStat{nums[0], nums[1], nums[2], nums[3], nums[4], nums[5], nums[6]}, nil
	}
	return cpuStat{}, fmt.Errorf("cpu line not found in /proc/stat")
}

func calcCPUPercent(prev, curr cpuStat) float64 {
	prevTotal := prev.user + prev.nice + prev.system + prev.idle + prev.iowait + prev.irq + prev.softirq
	currTotal := curr.user + curr.nice + curr.system + curr.idle + curr.iowait + curr.irq + curr.softirq
	prevIdle := prev.idle + prev.iowait
	currIdle := curr.idle + curr.iowait
	totalDiff := float64(currTotal - prevTotal)
	idleDiff := float64(currIdle - prevIdle)
	if totalDiff == 0 {
		return 0
	}
	return (totalDiff - idleDiff) / totalDiff * 100
}

func readMemoryPercent() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	vals := map[string]uint64{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			vals[key] = val
		}
	}

	total := vals["MemTotal"]
	available := vals["MemAvailable"]
	if total == 0 {
		return 0, fmt.Errorf("MemTotal not found")
	}
	used := total - available
	return float64(used) / float64(total) * 100, nil
}

func readDiskPercent(path string) (float64, error) {
	// Use syscall.Statfs for disk usage
	var stat syscallStatfs
	if err := statfs(path, &stat); err != nil {
		return 0, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total == 0 {
		return 0, nil
	}
	used := total - free
	return float64(used) / float64(total) * 100, nil
}

func readNetworkStat() (networkStat, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return networkStat{}, err
	}
	defer f.Close()

	var totalRx, totalTx uint64
	scanner := bufio.NewScanner(f)
	// Skip 2 header lines
	scanner.Scan()
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 10 {
			continue
		}
		iface := strings.TrimSuffix(fields[0], ":")
		if iface == "lo" {
			continue
		}
		rx, _ := strconv.ParseUint(fields[1], 10, 64)
		tx, _ := strconv.ParseUint(fields[9], 10, 64)
		totalRx += rx
		totalTx += tx
	}
	return networkStat{totalRx, totalTx}, nil
}
