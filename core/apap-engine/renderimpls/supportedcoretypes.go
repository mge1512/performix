// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package renderimpls

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Arm-Debug/apap-cli/apap-engine/cdf"
	"github.com/Arm-Debug/apap-cli/apap-engine/render"
	"github.com/Arm-Debug/apap-cli/apap-engine/run"
	"github.com/Arm-Debug/apap-cli/apap-engine/telemetry"
)

const supportedCoreTypesOutputSchemaVersion = "0.1"

var supportedCoreTypesComponentType = cdf.ComponentType{
	Name:          "target-info-supported-core-types",
	SchemaVersion: supportedCoreTypesOutputSchemaVersion,
}

// SupportedCoreTypes renderer exposes the collected CPU rows for which the
// engine has an embedded telemetry specification. It is configured explicitly
// by recipes that need this data, so existing TargetInfoRenderer consumers are
// unaffected.
type SupportedCoreTypes struct {
	config *render.Config
}

func (renderer *SupportedCoreTypes) Name() string {
	return "SupportedCoreTypes"
}

func (renderer *SupportedCoreTypes) Version() string {
	return "0.1.0"
}

func (renderer *SupportedCoreTypes) Configure(config *render.Config) error {
	renderer.config = config
	return nil
}

func (renderer *SupportedCoreTypes) GetInputSpec() render.InputSpec {
	return render.InputSpec{PortList: render.PortList{Ports: []render.PortSpec{
		{
			Name:          "target_info_cpus",
			Cardinality:   render.CardinalityPerRun,
			ComponentType: cdf.ComponentType{Name: "target-info-cpus", SchemaVersion: cpusOutputSchemaVersion},
		},
	}}}
}

func (renderer *SupportedCoreTypes) GetOutputSpec() render.OutputSpec {
	return render.OutputSpec{PortList: render.PortList{Ports: []render.PortSpec{
		{
			Name:          "supported_core_types",
			Cardinality:   render.CardinalityPerRun,
			ComponentType: supportedCoreTypesComponentType,
		},
	}}}
}

func createSupportedCoreTypesTable(
	ctx context.Context,
	conn *sql.Conn,
	sourceTable string,
	outputTable string,
) error {
	supportedModels := telemetry.SupportedCPUModels()
	placeholders := make([]string, len(supportedModels))
	args := make([]any, len(supportedModels))
	for i, model := range supportedModels {
		placeholders[i] = "?"
		args[i] = model
	}

	predicate := "FALSE"
	if len(placeholders) > 0 {
		predicate = fmt.Sprintf("name IN (%s)", strings.Join(placeholders, ", "))
	}

	// #nosec G201 -- table names come from manifest-generated identifiers; values are parameterized.
	query := fmt.Sprintf(`
		CREATE TABLE %s AS
		SELECT
			CAST(core_number AS BIGINT) AS core_number,
			CAST(cluster_id AS BIGINT) AS cluster_id,
			CAST(midr AS VARCHAR) AS midr,
			CAST(name AS VARCHAR) AS name
		FROM %s
		WHERE %s
	`,
		QuoteColumnName(outputTable),
		QuoteColumnName(sourceTable),
		predicate,
	)
	if _, err := conn.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to create supported core types table from '%s': %w", sourceTable, err)
	}
	return nil
}

func (renderer *SupportedCoreTypes) Initialize(
	session render.Session,
	resolvedDataSources map[string][]render.TableRef,
) error {
	inputTables, ok := resolvedDataSources["target_info_cpus"]
	if !ok || len(inputTables) == 0 {
		return fmt.Errorf("missing required input 'target_info_cpus'")
	}
	if len(inputTables) != len(session.Content().Entries) {
		return fmt.Errorf(
			"expected one 'target_info_cpus' input per run: got %d inputs for %d runs",
			len(inputTables),
			len(session.Content().Entries),
		)
	}

	for i, entry := range session.Content().Entries {
		outputTable := session.Manifest().AddEntry(render.NewManifestEntryInfo(
			supportedCoreTypesComponentType,
			renderer.config.Identity,
			[]run.RunID{entry.ID},
		))
		if err := createSupportedCoreTypesTable(
			context.Background(),
			session.Database().Conn,
			inputTables[i].Name,
			outputTable,
		); err != nil {
			return err
		}
	}

	return nil
}
