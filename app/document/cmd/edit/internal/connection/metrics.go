package connection

import "github.com/prometheus/client_golang/prometheus"

var (
	// connectionGauge 当前在线连接数, 机器人连接同样计入
	connectionGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mutidoc",
		Subsystem: "edit",
		Name:      "active_connections",
		Help:      "当前在线连接数, 包含用户与机器人",
	})

	// otGauge 当前内存中的 OT 文档会话数
	otGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mutidoc",
		Subsystem: "edit",
		Name:      "active_ot_sessions",
		Help:      "当前内存中的 OT 文档会话数",
	})
)

func init() {
	prometheus.MustRegister(connectionGauge, otGauge)
}
