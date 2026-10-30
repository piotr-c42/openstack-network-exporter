// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Red Hat

package ovnrelay

import (
	"context"
	"errors"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/appctl"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "ovnrelay"
}

func (Collector) Metrics() []lib.Metric {
	res := make([]lib.Metric, 0, len(metrics)+len(memoryMetrics))
	res = append(res, metrics...)
	for _, metric := range memoryMetrics {
		res = append(res, metric)
	}
	return res
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

func emit(ch chan<- prometheus.Metric, metric lib.Metric, value float64, labels ...string) {
	if !config.MetricSets().Has(metric.Set) {
		return
	}
	ch <- prometheus.MustNewConstMetric(metric.Desc(), metric.ValueType, value, labels...)
}

var memoryRe = regexp.MustCompile(`([a-zA-Z0-9_-]+):([0-9]+)`)

func parseMemory(output string) map[string]float64 {
	values := map[string]float64{}
	for _, match := range memoryRe.FindAllStringSubmatch(output, -1) {
		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}
		values[match[1]] = value
	}
	return values
}

func countRemotes(output string) float64 {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return float64(count)
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, err := ovsdb.RelayStatus(ctx)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Socket absent: the relay is not running or not deployed here.
			return
		}
		log.Debugf("relay status unavailable: %s", err)
		emit(ch, relayUp, 0)
		return
	}

	emit(ch, relayUp, 1)
	if status.Connected {
		emit(ch, relayUpstreamConnected, 1)
	} else {
		emit(ch, relayUpstreamConnected, 0)
	}
	emit(ch, relayDatabaseInfo, 1, status.Name, status.Model)

	if buf := appctl.OvsDbRelay("memory/show"); buf != "" {
		memory := parseMemory(buf)
		for key, metric := range memoryMetrics {
			if value, ok := memory[key]; ok {
				emit(ch, metric, value)
			}
		}
	}

	if buf := appctl.OvsDbRelay("ovsdb-server/list-remotes"); buf != "" {
		emit(ch, relayRemotes, countRemotes(buf))
	}
}
