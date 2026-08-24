// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

// @ts-check

/**
 * @typedef {import("../docs/jsdocs").RunComponentDescription} RunComponentDescription
 */

const TIMELINE_COUNTER_DATA_RELATIVE_PATH_PATTERN =
  /^(.+)\/key_type=(\d+)\/series_id=(\d+)\/bin_duration=(\d+)\/counter\.parquet$/i;

const sql = (strings, ...values) => {
  const text = String.raw({ raw: strings }, ...values).trim();
  const lines = text.split('\n');
  const indentedLines = lines.slice(1).filter((line) => line.trim().length > 0);
  const indentation =
    indentedLines.length === 0
      ? 0
      : Math.min(...indentedLines.map((line) => line.match(/^ */)[0].length));

  return [
    lines[0],
    ...lines
      .slice(1)
      .filter((line) => line.trim().length > 0)
      .map((line) => line.slice(indentation)),
  ].join('\n');
};

/**
 * @param {string} componentRelativePath
 * @returns {{
 *   componentRelativePath: string,
 *   rawSeriesKey: string,
 *   keyType: number,
 *   seriesId: number,
 *   binDuration: number,
 * } | null}
 */
function parseTimelineCounterDataPath(componentRelativePath) {
  const relativePath = String(componentRelativePath ?? '');
  const match = relativePath.match(TIMELINE_COUNTER_DATA_RELATIVE_PATH_PATTERN);
  if (!match) {
    return null;
  }

  const keyType = Number(match[2]);
  const seriesId = Number(match[3]);
  const binDuration = Number(match[4]);
  if (
    !Number.isSafeInteger(keyType) ||
    keyType < 0 ||
    !Number.isSafeInteger(seriesId) ||
    seriesId < 0 ||
    !Number.isSafeInteger(binDuration) ||
    binDuration <= 0
  ) {
    return null;
  }

  return {
    componentRelativePath: relativePath,
    rawSeriesKey: `key_${keyType}_series_${seriesId}`,
    keyType,
    seriesId,
    binDuration,
  };
}

/**
 * @param {import("../docs/jsdocs").RenderExecutionContext} context
 * @param {number} runIndex
 * @param {string} counterParquetPattern
 * @returns {{
 *   component: RunComponentDescription,
 *   componentPattern: string,
 *   componentRelativePath: string,
 *   rawSeriesKey: string,
 *   keyType: number,
 *   seriesId: number,
 *   binDuration: number,
 * }[]}
 */
function findTimelineCounterBindings(context, runIndex, counterParquetPattern) {
  const bindings = [];
  const seenBindings = new Set();
  const components = context.listRunComponents(runIndex, counterParquetPattern);

  for (const component of components) {
    const componentPath = String(component?.relativePath ?? '');
    const parsedPath = parseTimelineCounterDataPath(componentPath);
    if (!parsedPath) {
      throw new Error(
        `Invalid timeline counter component path: ${componentPath}`,
      );
    }

    const bindingKey = `${parsedPath.keyType}:${parsedPath.seriesId}:${parsedPath.binDuration}`;
    if (seenBindings.has(bindingKey)) {
      continue;
    }

    seenBindings.add(bindingKey);
    bindings.push({
      component,
      componentPattern: counterParquetPattern,
      ...parsedPath,
    });
  }

  if (bindings.length === 0) {
    return [];
  }

  bindings.sort((left, right) => {
    return (
      left.binDuration - right.binDuration ||
      left.keyType - right.keyType ||
      left.seriesId - right.seriesId ||
      left.componentRelativePath.localeCompare(right.componentRelativePath)
    );
  });

  return bindings;
}

/**
 * @param {number} binDuration
 * @returns {string}
 */
function baseComponentTypeName(binDuration) {
  return `timeline-base-${binDuration}`;
}

/**
 * @param {number} binDuration
 * @returns {string}
 */
function baseOutputName(binDuration) {
  return `timeline_base_${binDuration}`;
}

/**
 * @param {number} binDuration
 * @param {string} rawSeriesKey
 * @returns {string}
 */
function seriesComponentTypeName(binDuration, rawSeriesKey) {
  return `timeline-source-${rawSeriesKey}-${binDuration}`;
}

/**
 * @param {number} binDuration
 * @param {string} rawSeriesKey
 * @returns {string}
 */
function seriesOutputName(binDuration, rawSeriesKey) {
  return `timeline_source_${rawSeriesKey}_${binDuration}`;
}

/**
 * @param {string} rendererIdPrefix
 * @param {string} rawSeriesKey
 * @param {number} binDuration
 * @returns {string}
 */
function seriesRendererID(rendererIdPrefix, rawSeriesKey, binDuration) {
  return `${rendererIdPrefix}_source_${rawSeriesKey}_${binDuration}`;
}

/**
 * @param {Object} args
 * @param {number} args.binDuration
 * @param {string} args.rendererIdPrefix
 * @param {string} args.componentPattern
 * @returns {any}
 */
function makeBaseScanRenderer({
  binDuration,
  rendererIdPrefix,
  componentPattern,
}) {
  return {
    type: 'SQL',
    id: `${rendererIdPrefix}_base_${binDuration}`,
    config: {
      sql: sql`
        SELECT
          start_timestamp,
          end_timestamp,
          device_no,
          thread,
          value,
          key_type,
          series_id,
          bin_duration
        FROM read_parquet({{path:${componentPattern}}}, hive_partitioning = true)
        WHERE bin_duration = ${binDuration}
      `,
      inputs: [],
      output: {
        name: baseOutputName(binDuration),
        description: `Timeline base scan view for bin duration ${binDuration}.`,
        cardinality: 'one',
        component_type: {
          name: baseComponentTypeName(binDuration),
          schema_version: '1.0',
        },
      },
    },
  };
}

/**
 * @param {Object} args
 * @param {number} args.binDuration
 * @param {string} args.rendererIdPrefix
 * @param {string} args.rawSeriesKey
 * @param {number} args.keyType
 * @param {number} args.seriesId
 * @returns {any}
 */
function makeSeriesRenderer({
  binDuration,
  rendererIdPrefix,
  rawSeriesKey,
  keyType,
  seriesId,
}) {
  const basePortName = baseOutputName(binDuration);
  const baseRendererId = `${rendererIdPrefix}_base_${binDuration}`;

  return {
    type: 'SQL',
    id: seriesRendererID(rendererIdPrefix, rawSeriesKey, binDuration),
    config: {
      sql: sql`
        -- Preserve compressed source intervals so the provider-authored range
        -- query can discard unrelated rows before expanding bins.
        SELECT
          CAST(base.start_timestamp AS BIGINT) AS start_timestamp,
          CAST(base.end_timestamp AS BIGINT) AS end_timestamp,
          base.series_id,
          CAST(base.bin_duration AS BIGINT) AS bin_duration,
          base.device_no,
          base.thread,
          base.value
        FROM {{table:${basePortName}}} AS base
        WHERE base.key_type = ${keyType}
          AND base.series_id = ${seriesId}
      `,
      inputs: [
        {
          name: basePortName,
          description: `Timeline base scan input for ${binDuration}.`,
          cardinality: 'one',
          component_type: {
            name: baseComponentTypeName(binDuration),
            schema_version: '1.0',
          },
        },
      ],
      data_source: {
        tables: {
          [basePortName]: [
            {
              renderer_id: baseRendererId,
              output: baseOutputName(binDuration),
            },
          ],
        },
      },
      output: {
        name: seriesOutputName(binDuration, rawSeriesKey),
        description: `Compressed source ${rawSeriesKey} for bin duration ${binDuration}.`,
        cardinality: 'one',
        component_type: {
          name: seriesComponentTypeName(binDuration, rawSeriesKey),
          schema_version: '1.0',
        },
      },
    },
  };
}

/**
 * Build a static timeline SQL view family with one base and compressed-series chain per bin duration.
 *
 * Contract by layer:
 * - Base scan: loads the source intervals without changing value semantics.
 * - Series sources: retain compressed intervals for one series and duration.
 * - Provider-authored presentation queries filter and expand only the requested
 *   range before aggregation and chart projection.
 *
 * @param {Object} args
 * @param {string} [args.rendererId]
 * @param {{
 *   component: RunComponentDescription,
 *   componentPattern: string,
 *   componentRelativePath: string,
 *   rawSeriesKey: string,
 *   keyType: number,
 *   seriesId: number,
 *   binDuration: number,
 * }[]} args.bindings
 * @returns {{
 *   renderers: any[],
 *   timelineSources: Array<{
 *     rawSeriesKey: string,
 *     keyType: number,
 *     rendererId: string,
 *     output: string,
 *     seriesId: number,
 *     binDuration: number,
 *   }>,
 * }}
 */
function buildTimelineSQLRendererBundle(args) {
  const rendererId = args.rendererId ?? 'timeline_sql';
  const bindings = args.bindings ?? [];
  if (bindings.length === 0) {
    return { renderers: [], timelineSources: [] };
  }
  const renderers = [];
  const timelineSources = [];
  const baseRenderersByDuration = new Set();

  for (const binding of bindings) {
    if (!baseRenderersByDuration.has(binding.binDuration)) {
      renderers.push(
        makeBaseScanRenderer({
          binDuration: binding.binDuration,
          rendererIdPrefix: rendererId,
          componentPattern: binding.componentPattern,
        }),
      );
      baseRenderersByDuration.add(binding.binDuration);
    }

    const seriesRenderer = makeSeriesRenderer({
      binDuration: binding.binDuration,
      rendererIdPrefix: rendererId,
      rawSeriesKey: binding.rawSeriesKey,
      keyType: binding.keyType,
      seriesId: binding.seriesId,
    });
    renderers.push(seriesRenderer);

    timelineSources.push({
      rawSeriesKey: binding.rawSeriesKey,
      keyType: binding.keyType,
      rendererId: seriesRenderer.id,
      output: seriesOutputName(binding.binDuration, binding.rawSeriesKey),
      seriesId: binding.seriesId,
      binDuration: binding.binDuration,
    });
  }

  return {
    renderers,
    timelineSources,
  };
}

module.exports = {
  findTimelineCounterBindings,
  buildTimelineSQLRendererBundle,
};
