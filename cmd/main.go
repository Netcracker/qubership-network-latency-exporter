package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Netcracker/qubership-network-latency-exporter/pkg/collector"
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/logger"
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/metrics"
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/utils"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	versionCollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	prometheus.MustRegister(versionCollector.NewCollector("network_latency_exporter"))
}

func main() {
	var (
		webConfig    = webflag.AddFlags(kingpin.CommandLine, ":9273")
		packetsSent  = utils.GetEnvWithDefaultValue("PACKETS_NUM", "10")
		packetSize   = utils.GetEnvWithDefaultValue("PACKET_SIZE", "1500")
		protocolsStr = utils.GetEnvWithDefaultValue("CHECK_TARGET", "ICMP")
		probeTimeout = utils.GetEnvWithDefaultValue("REQUEST_TIMEOUT", "3")
		latencyTypes = utils.GetEnvWithDefaultValue("LATENCY_TYPES", "node_collector")
		metricsPath  = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		maxRequests = kingpin.Flag(
			"web.max-requests",
			"Maximum number of parallel scrape requests. Use 0 to disable.",
		).Default("40").Int()
	)

	promsLogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promsLogConfig)
	kingpin.Version(version.Print("network_latency_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Create a new slog logger with configured level and format
	log := promslog.New(promsLogConfig)

	_ = os.Setenv("LOG_LEVEL", logger.GetLogLevel().String())

	log.Info("Starting network_latency_exporter", "version", version.Info())
	log.Info("Build context", "context", version.BuildContext())

	namespace := utils.GetNamespace()
	log.Info("Namespace", "namespace", namespace)

	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := context.WithValue(baseCtx, collector.ContextKey, "main")

	rCfg, _ := ctrl.GetConfig()
	var clientSet *kubernetes.Clientset
	if rCfg != nil {
		clientSet = kubernetes.NewForConfigOrDie(rCfg)
	}

	var checkTargets []*metrics.CheckTarget
	for _, p := range strings.Split(strings.TrimSpace(protocolsStr), ",") {
		protocolAndPort := strings.Split(p, ":")
		checkTarget := &metrics.CheckTarget{}
		if protocolAsFlag, ok := collector.ProtocolToMtrFlag[protocolAndPort[0]]; ok {
			checkTarget.Protocol = protocolAndPort[0]
			checkTarget.MtrKey = protocolAsFlag
		} else {
			log.Warn("Skip incorrect or unsupported protocol", "protocol", p)
			continue
		}
		if len(protocolAndPort) == 2 {
			checkTarget.Port = protocolAndPort[1]
		} else {
			checkTarget.Port = "1"
		}
		checkTargets = append(checkTargets, checkTarget)
	}

	targets := collector.Discover(log)
	if targets != nil {
		targets = utils.ValidateTargets(log, targets)
		latencies := strings.Split(latencyTypes, ",")
		cfgCont := collector.NewConfigContainer(latencies, namespace, log)
		if err := cfgCont.Initialize(ctx, packetsSent, packetSize, probeTimeout, checkTargets, *targets, *metricsPath); err != nil {
			log.Error("Initialization failed", "err", err)
			os.Exit(1)
		}

		var enabledCollectors []collector.Collector
		for collectorName, enabled := range collector.GetCollectorStates() {
			if _, found := cfgCont.CollectorConfigs[string(collector.AsType(collectorName))]; found && enabled {
				log.Info("Collector enabled from main", "collector", collectorName)
				c, err := collector.GetCollector(collectorName, log)
				if err != nil {
					log.Error("Couldn't get collector", "collector", collectorName, "err", err)
					continue
				}
				enabledCollectors = append(enabledCollectors, c)
			}
		}

		for _, coll := range enabledCollectors {
			if cfg := cfgCont.GetConfig(ctx, coll.Type()); cfg != nil {
				err := coll.Initialize(ctx, cfg)
				if err != nil {
					log.Error("Can't initialize collector", "collector", coll.Name(), "err", err)
				}
			}
		}
		exporter := collector.New(ctx, collector.NewMetrics(), enabledCollectors, log)
		cfgCont.Exporter = exporter

		watcher, err := clientSet.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error("Failed to create node watcher", "err", err)
			syscall.Exit(1)
		}

		defer func(watcher watch.Interface) {
			watcher.Stop()
		}(watcher)

		nw := &nodeWatcher{
			ctx:    ctx,
			logger: log,
		}

		go nw.watch(log, watcher, cfgCont, ctx)

		// register exporter only once
		err = prometheus.Register(exporter)
		if err != nil {
			if !prometheus.Unregister(exporter) {
				log.Error("Exporter can't be unregistered")
				return
			}
			prometheus.MustRegister(exporter)
		}

		metricHandlerFunc := collector.MetricHandler(exporter, *maxRequests, log)
		http.Handle(*metricsPath, utils.AddHSTSHeader(promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, metricHandlerFunc)))
		http.Handle("/-/ready", utils.AddHSTSHeader(readinessChecker()))
		http.Handle("/-/healthy", utils.AddHSTSHeader(healthChecker()))

		srvBaseCtx := context.WithValue(context.Background(), collector.ContextKey, "http")
		srv := &http.Server{
			BaseContext: func(_ net.Listener) context.Context {
				return srvBaseCtx
			},
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
		}
		sd := &shutdown{
			srv:     srv,
			logger:  log,
			ctx:     context.WithValue(context.Background(), collector.ContextKey, "shutdown"),
			timeout: 30 * time.Second,
		}
		go sd.listen()
		log.Info("Starting server", "address", srv.Addr)
		exit := web.ListenAndServe(srv, webConfig, logger.NewLogger())

		cancel()
		if !errors.Is(exit, http.ErrServerClosed) {
			log.Error("Failed to start application", "err", exit)
		}
		log.Info("Server is shut down")
	} else {
		log.Info("Discovery is disabled")
	}
}

func healthChecker() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		},
	)
}

func readinessChecker() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("OK"))
		})
}
