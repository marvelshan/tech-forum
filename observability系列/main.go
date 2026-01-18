package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var (
	// Counter: 累計值（只增不減）
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_request_total",
			Help: "Total number of requests by user and method",
		},
		[]string{"user", "method"},
	)

	// Gauge: 可增可減的即時值
	activeConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_active_connections",
			Help: "Current active connections by service",
		},
		[]string{"service"},
	)

	// Histogram: 分佈統計（例如請求延遲）
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 10), // 0.1 ~ 1.0s
		},
		[]string{"endpoint"},
	)
	cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_cpu_usage_percent",
		Help: "Current CPU usage percentage",
	})
	memoryUsedBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_memory_used_bytes",
		Help: "Current memory used in bytes",
	})
	memoryTotalBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_memory_total_bytes",
		Help: "Total memory in bytes",
	})
	load1 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_load1",
		Help: "1-minute load average",
	})
	diskUsedBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_disk_used_bytes",
		Help: "Disk usage in bytes (root partition)",
	})
	diskTotalBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_disk_total_bytes",
		Help: "Total disk space in bytes (root partition)",
	})
	networkRxBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_network_receive_bytes_total",
		Help: "Total network received bytes",
	})
	networkTxBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_network_transmit_bytes_total",
		Help: "Total network transmitted bytes",
	})
	swapTotalBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_swap_total_bytes",
		Help: "Total swap space in bytes",
	})
	swapUsedBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_swap_used_bytes",
		Help: "Used swap space in bytes",
	})
	swapFreeBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_swap_free_bytes",
		Help: "Free swap space in bytes",
	})
)

func init() {
	prometheus.MustRegister(requestCount)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsedBytes)
	prometheus.MustRegister(memoryTotalBytes)
	prometheus.MustRegister(load1)
	prometheus.MustRegister(diskUsedBytes)
	prometheus.MustRegister(diskTotalBytes)
	prometheus.MustRegister(networkRxBytes)
	prometheus.MustRegister(networkTxBytes)
	prometheus.MustRegister(swapTotalBytes)
	prometheus.MustRegister(swapUsedBytes)
	prometheus.MustRegister(swapFreeBytes)
}

func simulateMetrics() {
	// 模擬動態數據
	rand.Seed(time.Now().UnixNano())

	// 初始化一些值
	requestCount.WithLabelValues("admin", "GET").Add(rand.Float64())
	requestCount.WithLabelValues("guest", "GET").Add(rand.Float64())
	requestCount.WithLabelValues("admin", "POST").Add(rand.Float64())

	activeConnections.WithLabelValues("api").Set(42 + rand.Float64()*20)
	activeConnections.WithLabelValues("db").Set(12 + rand.Float64()*10)

	// 模擬隨機延遲
	for i := 0; i < 100; i++ {
		duration := 0.1 + rand.Float64()*0.9 // 0.1 ~ 1.0 秒
		endpoint := []string{"/api/users", "/api/orders"}[rand.Intn(2)]
		requestDuration.WithLabelValues(endpoint).Observe(duration)
	}

	time.Sleep(1 * time.Millisecond)
}
func collectHostMetrics() {
	for {
		// CPU 使用率（非阻塞，需 time.Sleep）
		percent, err := cpu.Percent(time.Second, false)
		if err == nil && len(percent) > 0 {
			cpuUsage.Set(percent[0])
		}

		// 記憶體
		vmStat, err := mem.VirtualMemory()
		if err == nil {
			memoryUsedBytes.Set(float64(vmStat.Used))
			memoryTotalBytes.Set(float64(vmStat.Total))
		}

		// 系統負載
		loadStat, err := load.Avg()
		if err == nil {
			load1.Set(loadStat.Load1)
		}

		// 磁碟（根目錄）
		diskStat, err := disk.Usage("/")
		if err == nil {
			diskUsedBytes.Set(float64(diskStat.Used))
			diskTotalBytes.Set(float64(diskStat.Total))
		}

		// 網路（合併所有介面）
		netStats, err := net.IOCounters(true)
		if err == nil {
			var rx, tx uint64
			for _, stat := range netStats {
				rx += stat.BytesRecv
				tx += stat.BytesSent
			}
			networkRxBytes.Set(float64(rx))
			networkTxBytes.Set(float64(tx))
		}

		// Swap 使用量
		swapStat, err := mem.SwapMemory()
		if err == nil {
			swapTotalBytes.Set(float64(swapStat.Total))
			swapUsedBytes.Set(float64(swapStat.Used))
			swapFreeBytes.Set(float64(swapStat.Free))
		}

		time.Sleep(5 * time.Second) // 每 5 秒更新一次
	}
}

func main() {
	simulateMetrics()
	go collectHostMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8050", nil)
}
