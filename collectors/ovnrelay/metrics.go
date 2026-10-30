// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Red Hat

package ovnrelay

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var relayUp = lib.Metric{
	Name:        "ovn_sb_relay_up",
	Description: "1 if the local OVSDB relay database is reachable; absent when the relay socket does not exist",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var relayUpstreamConnected = lib.Metric{
	Name:        "ovn_sb_relay_upstream_connected",
	Description: "1 if the relay has an active connection to an OVN Southbound server, 0 otherwise; absent when the relay socket does not exist",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var relayDatabaseInfo = lib.Metric{
	Name:        "ovn_sb_relay_database_info",
	Description: "Constant 1 with the relay database name and service model",
	Labels:      []string{"database", "model"},
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var relayRemotes = lib.Metric{
	Name:        "ovn_sb_relay_remotes",
	Description: "Number of OVSDB remotes configured on the relay process",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var relayMonitors = lib.Metric{
	Name:        "ovn_sb_relay_monitors",
	Description: "Number of active monitor sessions on the relay database",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_PERF,
}

var relayAtoms = lib.Metric{
	Name:        "ovn_sb_relay_atoms",
	Description: "Number of atoms in the relay database",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_PERF,
}

var relayCells = lib.Metric{
	Name:        "ovn_sb_relay_cells",
	Description: "Number of cells in the relay database",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_PERF,
}

var relayWeakRefs = lib.Metric{
	Name:        "ovn_sb_relay_weak_refs",
	Description: "Number of weak references in the relay database",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_PERF,
}

var relayTxnHistory = lib.Metric{
	Name:        "ovn_sb_relay_txn_history",
	Description: "Number of transactions in the relay transaction history",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_PERF,
}

var relayTxnHistoryAtoms = lib.Metric{
	Name:        "ovn_sb_relay_txn_history_atoms",
	Description: "Number of atoms in the relay transaction history",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_PERF,
}

// memoryMetrics maps the keys printed by "memory/show" to their metric.
var memoryMetrics = map[string]lib.Metric{
	"monitors":          relayMonitors,
	"atoms":             relayAtoms,
	"cells":             relayCells,
	"n-weak-refs":       relayWeakRefs,
	"txn-history":       relayTxnHistory,
	"txn-history-atoms": relayTxnHistoryAtoms,
}

var metrics = []lib.Metric{
	relayUp,
	relayUpstreamConnected,
	relayDatabaseInfo,
	relayRemotes,
}
