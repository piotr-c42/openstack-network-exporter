// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package appctl

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
)

type appctlDaemon string

const (
	ovsVswitchd   appctlDaemon = "ovs-vswitchd"
	ovnController appctlDaemon = "ovn-controller"
	ovnNorthd     appctlDaemon = "ovn-northd"
	ovsDbServer   appctlDaemon = "ovsdb-server"
	ovsDbRelay    appctlDaemon = "ovsdb-server-relay"
)

func getPidFromFile(pidfile string) (int, error) {
	f, err := os.Open(pidfile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(buf)))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

func getPidFromCtlFiles(rundir string, daemon appctlDaemon) (int, error) {
	// Look for .ctl files matching the daemon pattern
	pattern := fmt.Sprintf("%s.*.ctl", daemon)
	matches, err := filepath.Glob(filepath.Join(rundir, pattern))
	if err != nil {
		return 0, err
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no control socket files found for %s", daemon)
	}

	// Extract PID from the first matching file
	// Expected format: daemon.pid.ctl
	re := regexp.MustCompile(fmt.Sprintf(`%s\.(\d+)\.ctl$`, regexp.QuoteMeta(string(daemon))))
	for _, match := range matches {
		basename := filepath.Base(match)
		submatch := re.FindStringSubmatch(basename)
		if len(submatch) == 2 {
			pid, err := strconv.Atoi(submatch[1])
			if err == nil {
				return pid, nil
			}
		}
	}

	return 0, fmt.Errorf("could not extract PID from control socket files for %s", daemon)
}

func prepareCallDbServer(method string, rundir string, args ...string) (string, []string, error) {

	const (
		ovnsbDb       = "OVN_Southbound"
		ovnnbDb       = "OVN_Northbound"
		clusterStatus = "cluster/status"
	)

	// Check which socket file exists
	sbSocket := filepath.Join(rundir, "ovnsb_db.ctl")
	nbSocket := filepath.Join(rundir, "ovnnb_db.ctl")

	var sockpath string
	var dbName string

	if _, err := os.Stat(sbSocket); err == nil {
		sockpath = sbSocket
		dbName = ovnsbDb
	} else if _, err := os.Stat(nbSocket); err == nil {
		sockpath = nbSocket
		dbName = ovnnbDb
	} else {
		return "", args, fmt.Errorf("no control socket files found for the ovs db server")
	}

	if method == clusterStatus {
		args = []string{dbName}
	}

	return sockpath, args, nil
}

func call(daemon appctlDaemon, method string, args ...string) string {
	var rundir, sockpath string
	var err error

	switch daemon {
	case ovsVswitchd:
		rundir = config.OvsRundir()
	case ovnController:
		rundir = config.OvnRundir()
	case ovnNorthd:
		rundir = config.OvnRundir()
	case ovsDbServer:
		rundir = config.OvsdbRundir()
	case ovsDbRelay:
		rundir = config.OvsdbRelayRundir()
	default:
		panic(fmt.Errorf("unknown daemon value: %v", daemon))
	}

	if daemon == ovsDbRelay {
		sockpath = filepath.Join(rundir, "ovnsb_relay_db.ctl")
	} else if daemon == ovsDbServer {
		sockpath, args, err = prepareCallDbServer(method, rundir, args...)
		if err != nil {
			log.Errf("Failed to prepare call to %s: %s", daemon, err)
			return ""
		}
	} else {
		pidfile := filepath.Join(rundir, fmt.Sprintf("%s.pid", daemon))

		// First try to get PID from .pid file
		pid, err := getPidFromFile(pidfile)
		if err != nil {
			log.Debugf("Failed to read PID file %s: %s, trying to find PID from .ctl files", pidfile, err)
			// If that fails, try to extract PID from .ctl files
			pid, err = getPidFromCtlFiles(rundir, daemon)
			if err != nil {
				log.Errf("Failed to get PID for %s: %s", daemon, err)
				return ""
			}
		}

		sockpath = filepath.Join(rundir, fmt.Sprintf("%s.%d.ctl", daemon, pid))
	}

	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		log.Errf("net.Dial: %s", err)
		return ""
	}

	client := rpc.NewClientWithCodec(NewClientCodec(conn))
	defer func() {
		err := client.Close()
		if err != nil {
			log.Warningf("close: %s", err)
		}
	}()

	if args == nil {
		args = make([]string, 0)
	}

	var reply string

	log.Debugf("calling: %s %s", method, args)
	if err = client.Call(method, args, &reply); err != nil {
		log.Errf("call(%s): %s", method, err)
		return ""
	}

	return reply
}

func OvsVSwitchd(method string, args ...string) string {
	return call(ovsVswitchd, method, args...)
}

func OvnController(method string, args ...string) string {
	return call(ovnController, method, args...)
}

func OvnNorthd(method string, args ...string) string {
	return call(ovnNorthd, method, args...)
}

func OvsDbServer(method string, args ...string) string {
	return call(ovsDbServer, method, args...)
}

// OvsDbRelay calls an appctl method on the OVSDB relay process.
func OvsDbRelay(method string, args ...string) string {
	return call(ovsDbRelay, method, args...)
}
