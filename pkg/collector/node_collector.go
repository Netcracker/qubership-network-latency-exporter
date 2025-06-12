package collector

import (
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Netcracker/network-latency-exporter/pkg/metrics"
	"github.com/Netcracker/network-latency-exporter/pkg/model"
	"github.com/Netcracker/network-latency-exporter/pkg/utils"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ProtocolToMtrFlag = map[string]string{
		"UDP":  "--udp",
		"ICMP": "",
		"TCP":  "--tcp",
	}
	nodeConfig model.NodeCollector
	help       = map[string]string{
		"_status":     "Status of network latency",
		"_sent":       "Packets sent",
		"_received":   "Packets received",
		"_rtt_mean":   "Average mean of packets RTT",
		"_rtt_min":    "Best round trip time",
		"_rtt_max":    "Worst round trip time",
		"_rtt_stddev": "Standard deviation of packets mean RTT",
		"_hops_num":   "Number of hops in packet path",
	}
)

type NodeCollector struct {
	Desc         *prometheus.Desc
	ValueType    prometheus.ValueType
	Logger       *slog.Logger
	PacketsSent  string
	PacketSize   string
	ProbeTimeout string
	CheckTargets []*metrics.CheckTarget
	Targets      metrics.PingHostList
}

func init() {
	registerCollector(string(NodeType), defaultEnabled, newNodeCollector)
}

func (nodeCollector *NodeCollector) Close() {
	//nodeCollector.Resources = nodeCollector.Resources[:0]
}

func newNodeCollector(logger *slog.Logger) (Collector, error) {
	return &NodeCollector{
		Desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", NodeType.String()),
			"Network latency metrics for nodes",
			nil, nil),
		ValueType: prometheus.GaugeValue,
		Logger:    logger,
	}, nil
}

func (nodeCollector *NodeCollector) Initialize(ctx context.Context, config interface{}) error {
	cfg := reflect.ValueOf(config)
	switch cfg.Kind() {
	case reflect.Struct:
		nodeConfig = config.(model.NodeCollector)
	default:
		return errors.Errorf("Unsupported type: %v", cfg.Type())
	}
	return nil
}

func (nodeCollector *NodeCollector) Scrape(ctx context.Context, mets *Metrics, ch chan<- prometheus.Metric) error {
	var m []*metrics.NetworkLatencyMetric
	// Command line args to run MTR
	mtrArgs := []string{
		"-G", // timeout for probe
		nodeConfig.ProbeTimeout,
		"-Z", // how long keep probe socket open
		nodeConfig.ProbeTimeout,
		"-n",     // print destination as IP address
		"--json", // output format
		"-s",     // packet size in bytes
		nodeConfig.PacketSize,
		"-c", // packets count to sent
		nodeConfig.PacketsSent,
	}

	// Prepare multi-threaded execution
	var wg sync.WaitGroup
	wg.Add(len(nodeConfig.Targets.Targets) * len(nodeConfig.CheckTargets)) // how many gorutines need to wait before ending
	var execErr error                                                      // to propagate error from a separated thread to the main thread
	var mu sync.Mutex

	//MTR takes approx 1 second for each packet sent
	packets, er := strconv.Atoi(nodeConfig.PacketsSent)
	if er != nil {
		nodeCollector.Logger.Error("Packets Sent has incorrect value", "value", nodeConfig.PacketsSent)
	}

	mtrTimeout := utils.GetEnvWithDefaultValue("MTR_TIMEOUT", "10")
	extraTimeout, err := strconv.Atoi(mtrTimeout)
	if err != nil {
		nodeCollector.Logger.Error("Error while converting timeout value", "err", err)
	}
	timeout := (time.Duration(packets + extraTimeout)) * time.Second

	// Collect metrics
	for _, tgt := range nodeConfig.Targets.Targets {
		// Execute mtr for each protocol in separate gorutine
		for _, protocol := range nodeConfig.CheckTargets {
			nodeCollector.Logger.Debug("Executing protocol on target", "protocol", protocol, "target", tgt.Name)
			go func(t metrics.PingHost, p *metrics.CheckTarget) {
				defer wg.Done()
				// Prepare arguments for mtr
				args := make([]string, len(mtrArgs))
				copy(args, mtrArgs)
				args = append(args, p.MtrKey)
				args = append(args, "-P")
				args = append(args, p.Port)
				args = append(args, t.IPAddress)

				start := time.Now()
				nodeCollector.Logger.Debug("Executing mtr", "args", args)

				//Build context timeout with 10 seconds extra
				ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				// Execute mtr
				output, err := exec.CommandContext(ctxTimeout, "mtr", args...).Output()
				if err != nil {
					nodeCollector.Logger.Error("Failed to run mtr process", "err", err)
					execErr = err
				}
				if ctxTimeout.Err() == context.DeadlineExceeded {
					nodeCollector.Logger.Error("Process timeout")
					execErr = ctxTimeout.Err()
				}

				// Parse output
				mtrOutput := &metrics.MtrOutput{}
				err = json.Unmarshal(output, mtrOutput)
				if err != nil {
					nodeCollector.Logger.Error("Error while unmarshalling mtr output", "err", err)
					execErr = err
				}
				end := time.Now()
				nodeCollector.Logger.Debug("MTR execution completed", "output", mtrOutput, "duration", end.Sub(start))

				// Transform to metric.
				// Read data from hop with host equals to target address.
				// If there is no such hop mark target as unreachable and set zero values.
				metric := metrics.NewNetworkLatencyMetric(t.Name, t.IPAddress, strings.ToUpper(p.Protocol), p.Port, nodeConfig.PacketsSent)
				metric.Fields.HopsNum = len(mtrOutput.Report.Hops)
				for _, hop := range mtrOutput.Report.Hops {
					if hop.Host == t.IPAddress {
						metric.Fields.Status = metrics.StatusOk // host has been reached
						// Fill measures
						metric.Fields.TotalReceived = metric.Fields.TotalSent - int(float64(metric.Fields.TotalSent)*(hop.Loss/100.0))
						metric.Fields.RttMean = hop.RttMean
						metric.Fields.RttMin = hop.RttMin
						metric.Fields.RttMax = hop.RttMax
						metric.Fields.RttDeviation = hop.RttDeviation
					}
				}
				if metric != nil {
					mu.Lock()
					defer mu.Unlock()
					m = append(m, metric)
				}
			}(tgt, protocol)
		}
	}

	wg.Wait()
	if execErr != nil {
		return execErr
	}

	nodeName := utils.GetEnvWithDefaultValue("NODE_NAME", "localhost")

	metric_names := []string{"_status", "_sent", "_received", "_rtt_mean", "_rtt_min", "_rtt_max", "_rtt_stddev", "_hops_num"}
	for _, met := range m {
		if met == nil {
			continue
		}
		labels := []string{"source", "destination", "destinationIp", "packets", "protocol", "port"}
		labelValues := []string{nodeName, met.Tags.Dest, met.Tags.DestIp, strconv.Itoa(met.Fields.TotalSent), met.Tags.Protocol, met.Tags.Port}
		for i, metricName := range metric_names {
			buildInfo := prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "network_latency" + metricName,
					Help: help[metricName],
				},
				labels,
			)

			switch i {
			case 0:
				buildInfo.WithLabelValues(labelValues...).Set(float64(met.Fields.Status))
			case 1:
				buildInfo.WithLabelValues(labelValues...).Set(float64(met.Fields.TotalSent))
			case 2:
				buildInfo.WithLabelValues(labelValues...).Set(float64(met.Fields.TotalReceived))
			case 3:
				value, _ := strconv.ParseFloat(strconv.FormatFloat(met.Fields.RttMean, 'f', 2, 64), 64)
				buildInfo.WithLabelValues(labelValues...).Set(value)
			case 4:
				value, _ := strconv.ParseFloat(strconv.FormatFloat(met.Fields.RttMin, 'f', 2, 64), 64)
				buildInfo.WithLabelValues(labelValues...).Set(value)
			case 5:
				value, _ := strconv.ParseFloat(strconv.FormatFloat(met.Fields.RttMax, 'f', 2, 64), 64)
				buildInfo.WithLabelValues(labelValues...).Set(value)
			case 6:
				value, _ := strconv.ParseFloat(strconv.FormatFloat(met.Fields.RttDeviation, 'f', 2, 64), 64)
				buildInfo.WithLabelValues(labelValues...).Set(value)
			default:
				buildInfo.WithLabelValues(labelValues...).Set(float64(met.Fields.HopsNum))
			}

			buildInfo.Collect(ch)
		}
	}
	return nil
}

func (nodeCollector *NodeCollector) Type() Type {
	return NodeType
}

// Name of the Scraper. Should be unique.
func (nodeCollector *NodeCollector) Name() string {
	return NodeType.String()
}
