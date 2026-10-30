// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Red Hat

package ovsdb

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/database/inmemory"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb/serverdb"
	"github.com/ovn-kubernetes/libovsdb/server"
)

func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ovnrelay")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func setRelayRundir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("OPENSTACK_NETWORK_EXPORTER_OVSDB_RELAY_RUNDIR", dir)
	if err := config.Parse(); err != nil {
		t.Fatal(err)
	}
}

func startRelayServer(t *testing.T, path string) *server.OvsdbServer {
	t.Helper()
	clientModel, err := serverdb.FullDatabaseModel()
	if err != nil {
		t.Fatal(err)
	}
	logger := logr.Discard()
	db := inmemory.NewDatabase(map[string]model.ClientDBModel{"_Server": clientModel}, &logger)
	dbModel, errs := model.NewDatabaseModel(serverdb.Schema(), clientModel)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	srv, err := server.NewOvsdbServer(db, &logger, dbModel)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve("unix", path) }()
	waitFor(t, 2*time.Second, "the relay server", srv.Ready)
	return srv
}

func seedRelayDatabase(t *testing.T, path string, connected bool) {
	t.Helper()
	clientModel, err := serverdb.FullDatabaseModel()
	if err != nil {
		t.Fatal(err)
	}
	logger := logr.Discard()
	db, err := client.NewOVSDBClient(
		clientModel,
		client.WithEndpoint("unix:"+path),
		client.WithLogger(&logger),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ops, err := db.Create(&serverdb.Database{
		Name:      "OVN_Southbound",
		Model:     serverdb.DatabaseModelRelay,
		Connected: connected,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Transact(ctx, ops...); err != nil {
		t.Fatal(err)
	}
}

func TestRelayStatus(t *testing.T) {
	tests := []struct {
		name      string
		connected bool
	}{
		{name: "connected", connected: true},
		{name: "disconnected", connected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := shortTempDir(t)
			setRelayRundir(t, dir)
			sock := filepath.Join(dir, "ovnsb_relay_db.sock")
			srv := startRelayServer(t, sock)
			defer srv.Close()
			seedRelayDatabase(t, sock, tt.connected)

			database, err := RelayStatus(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if database.Name != "OVN_Southbound" || database.Model != "relay" ||
				database.Connected != tt.connected {
				t.Fatalf("database = %+v, want OVN_Southbound/relay/connected=%v",
					database, tt.connected)
			}
		})
	}
}

func TestRelayStatusMissingDatabase(t *testing.T) {
	dir := shortTempDir(t)
	setRelayRundir(t, dir)
	srv := startRelayServer(t, filepath.Join(dir, "ovnsb_relay_db.sock"))
	defer srv.Close()

	_, err := RelayStatus(context.Background())
	if err == nil || !strings.Contains(err.Error(), "OVN_Southbound not found") {
		t.Fatalf("error = %v, want a missing OVN_Southbound error", err)
	}
}

func TestRelayStatusMissingSocket(t *testing.T) {
	setRelayRundir(t, shortTempDir(t))

	_, err := RelayStatus(context.Background())
	if err == nil {
		t.Fatal("expected an error when the relay socket is missing")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want fs.ErrNotExist", err)
	}
}
