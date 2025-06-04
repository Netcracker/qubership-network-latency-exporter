package collector

import (
	"context"
	"log/slog"

	"github.com/Netcracker/network-latency-exporter/pkg/metrics"
	"github.com/Netcracker/network-latency-exporter/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getClusterNodes returns list of cluster nodes.
func getClusterNodes() ([]corev1.Node, error) {
	// Creates the in-cluster client
	clientSet, err := utils.GetClientset()
	if err != nil {
		return nil, err
	}

	// Reads nodes list
	nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return nodes.Items, nil
}

func Discover(logger *slog.Logger) *metrics.PingHostList {
	if utils.GetEnvWithDefaultValue("DISCOVER_ENABLE", "true") == "true" {
		logger.Debug("Discovering cluster nodes as ping targets")
		rawNodes, err := getClusterNodes()
		if err != nil {
			logger.Error("Error getting cluster nodes", "err", err)
			return nil
		}

		// Parse nodes as targets
		targets := &metrics.PingHostList{}

		for _, n := range rawNodes {
			nodeAddress := ""
			nodeName := ""
			for _, a := range n.Status.Addresses {
				if a.Type == corev1.NodeInternalIP {
					nodeAddress = a.Address
				}
				if a.Type == corev1.NodeHostName {
					nodeName = a.Address
				}
			}

			if nodeAddress != "" {
				// Skip current node
				if nodeName != utils.GetEnvWithDefaultValue("NODE_NAME", "localhost") {
					logger.Debug("Discovered node", "ipAddress", nodeAddress, "name", nodeName)
					targets.Targets = append(targets.Targets, metrics.PingHost{IPAddress: nodeAddress, Name: nodeName})
				}
			}
		}
		return targets
	} else {
		logger.Info("Skip discovering. Script disabled.")
		return nil
	}
}
