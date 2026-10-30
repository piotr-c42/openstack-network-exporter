// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Red Hat

package ovsdb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
	"github.com/ovn-kubernetes/libovsdb/ovsdb/serverdb"
)

// RelayDatabase describes the local OVSDB relay database.
type RelayDatabase struct {
	Name      string
	Model     string
	Connected bool
}

// relayConnect opens a client to the relay _Server database.
func relayConnect(ctx context.Context) (client.Client, model.DatabaseModel, error) {
	endpoint := "unix:" + filepath.Join(config.OvsdbRelayRundir(), "ovnsb_relay_db.sock")

	clientModel, err := serverdb.FullDatabaseModel()
	if err != nil {
		return nil, model.DatabaseModel{}, fmt.Errorf("relay FullDatabaseModel: %w", err)
	}
	mod, errs := model.NewDatabaseModel(serverdb.Schema(), clientModel)
	if len(errs) > 0 {
		return nil, model.DatabaseModel{}, fmt.Errorf("relay model: %w", errs[len(errs)-1])
	}

	db, err := client.NewOVSDBClient(
		clientModel,
		client.WithEndpoint(endpoint),
		client.WithLogger(log.OvsdbLogger()),
	)
	if err != nil {
		return nil, model.DatabaseModel{}, fmt.Errorf("relay NewOVSDBClient: %w", err)
	}
	if err := db.Connect(ctx); err != nil {
		return nil, model.DatabaseModel{}, fmt.Errorf("relay connect: %w", err)
	}

	return db, mod, nil
}

// RelayStatus reads the Database row of the local OVSDB relay _Server database.
//
// The _Server database is a system database that is not part of the OVN
// Southbound schema. It is served by the relay process on its own unix socket.
// A missing socket is returned wrapped with fs.ErrNotExist.
func RelayStatus(ctx context.Context) (*RelayDatabase, error) {
	db, mod, err := relayConnect(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	info, err := mod.NewModelInfo(&serverdb.Database{})
	if err != nil {
		return nil, fmt.Errorf("relay NewModelInfo: %w", err)
	}

	results, err := db.Transact(ctx, ovsdb.Operation{
		Op:    ovsdb.OperationSelect,
		Table: info.Metadata.TableName,
	})
	if err != nil {
		return nil, fmt.Errorf("relay transact: %w", err)
	}

	for _, result := range results {
		if result.Error != "" {
			return nil, fmt.Errorf("relay transact: %s: %s", result.Error, result.Details)
		}
		for _, row := range result.Rows {
			var database serverdb.Database
			info, _ = mod.NewModelInfo(&database)
			if err := mod.Mapper.GetRowData(&row, info); err != nil {
				return nil, fmt.Errorf("relay row: %w", err)
			}
			if database.Name == "OVN_Southbound" {
				return &RelayDatabase{
					Name:      database.Name,
					Model:     string(database.Model),
					Connected: database.Connected,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("OVN_Southbound not found in relay _Server.Database table")
}
