package model

import (
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/metrics"
	"k8s.io/client-go/kubernetes"
)

func init() {}

type PodCollector struct {
	ClientSet    kubernetes.Interface
	PacketsSent  string
	PacketSize   string
	ProbeTimeout string
	CheckTargets []*metrics.CheckTarget
	Targets      metrics.PingHostList
	MetricsPath  string
}
