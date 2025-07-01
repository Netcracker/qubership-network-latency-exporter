package main

import (
	"context"

	"log/slog"

	"github.com/Netcracker/qubership-network-latency-exporter/pkg/collector"
	"github.com/Netcracker/qubership-network-latency-exporter/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type nodeWatcher struct {
	ctx    context.Context
	logger *slog.Logger
}

func (fw *nodeWatcher) watch(logger *slog.Logger, watcher watch.Interface, cfgCont *collector.Container, ctx context.Context) {
	for event := range watcher.ResultChan() {
		logger.Info("Event occurred",
			"eventType", event.Type,
			"nodeName", event.Object.(*v1.Node).Name)
		if event.Type == watch.Added || event.Type == watch.Modified || event.Type == watch.Deleted {
			targets := collector.Discover(logger)
			if targets != nil {
				targets = utils.ValidateTargets(logger, targets)
				cfgCont.UpdateTargets(ctx, *targets)
			}
		}
	}
}
