// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

// @ts-check

const processStopGracePeriodMilliseconds = 1000;

/**
 * @typedef {{result: {exitCode: number}} | {error: unknown}} ProcessExitOutcome
 */

/**
 * @typedef {{
 *   handle: import("../recipes/docs/jsdocs").ProcessHandle | null,
 *   exitPromise: Promise<ProcessExitOutcome>
 * }} ProcessState
 */

/**
 * Create a process state whose exit promise resolves to a result or error,
 * clearing its handle only after a confirmed exit. A failed wait leaves the
 * handle available for cleanup.
 *
 * @param {{
 *   handle: import("../recipes/docs/jsdocs").ProcessHandle | null,
 *   exitPromise: Promise<{exitCode: number}> | null
 * }} processState
 * @returns {ProcessState}
 */
function promiseProcessExit(processState) {
  if (!processState.handle || !processState.exitPromise) {
    throw new Error(
      'Cannot promise a process exit without a handle and exit promise',
    );
  }

  /** @type {ProcessState} */
  const promisedProcess = {
    handle: processState.handle,
    exitPromise: processState.exitPromise.then(
      (result) => {
        promisedProcess.handle = null;
        return { result };
      },
      (error) => ({ error }),
    ),
  };
  return promisedProcess;
}

/**
 * @param {
 *   | ProcessExitOutcome
 *   | {timedOut: true}
 *   | {interruptFailed: true}
 * } outcome
 * @returns {
 *   | {state: 'interrupted', result: {exitCode: number}}
 *   | {state: 'killed'}
 *   | {state: 'error', error: unknown}
 * }
 */
function classifyProcessOutcome(outcome) {
  if ('result' in outcome) {
    return {
      state: /** @type {const} */ ('interrupted'),
      result: outcome.result,
    };
  }
  if ('error' in outcome) {
    return {
      state: /** @type {const} */ ('error'),
      error: outcome.error,
    };
  }
  return { state: /** @type {const} */ ('killed') };
}

/**
 * @param {import("../recipes/docs/jsdocs").Engine} engine
 * @param {ProcessState} processState
 * @returns {Promise<
 *   | {result: {exitCode: number}}
 *   | {error: unknown}
 *   | {timedOut: true}
 *   | {interruptFailed: true}
 * >}
 */
async function interruptAndWait(engine, processState) {
  const handle = processState.handle;
  if (!handle) {
    return await processState.exitPromise;
  }

  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let timeout;
  try {
    await handle.interrupt();
    return await Promise.race([
      processState.exitPromise,
      new Promise((resolve) => {
        timeout = setTimeout(
          () => resolve({ timedOut: /** @type {const} */ (true) }),
          processStopGracePeriodMilliseconds,
        );
      }),
    ]);
  } catch (error) {
    engine.log(
      'warn',
      `Failed to interrupt process ${handle.pid()}: ${error?.message ?? error}`,
    );
    return { interruptFailed: /** @type {const} */ (true) };
  } finally {
    if (timeout !== undefined) {
      clearTimeout(timeout);
    }
  }
}

/**
 * Interrupt a process and give it a bounded grace period to exit before
 * escalating to kill.
 *
 * Kill will succeed if the SIGKILL is sent, and we assume that the process has
 * exited, although this cannot be confirmed.
 *
 * @param {import("../recipes/docs/jsdocs").Engine} engine
 * @param {ProcessState} processState
 * @returns {Promise<{exitCode: number} | undefined>}
 */
async function endProcess(engine, processState) {
  const handle = processState.handle;
  if (!handle) {
    const outcome = classifyProcessOutcome(await processState.exitPromise);
    if (outcome.state === 'error') {
      throw outcome.error;
    }
    return outcome.result;
  }

  const outcome = classifyProcessOutcome(
    await interruptAndWait(engine, processState),
  );

  if (outcome.state === 'interrupted') {
    return outcome.result;
  }

  try {
    await handle.kill();
  } catch (error) {
    engine.log(
      'warn',
      `Failed to kill process ${handle.pid()}: ${error?.message ?? error}`,
    );
    throw error;
  }

  if (outcome.state === 'error') {
    throw outcome.error;
  }
}

module.exports = {
  classifyProcessOutcome,
  endProcess,
  interruptAndWait,
  promiseProcessExit,
};
