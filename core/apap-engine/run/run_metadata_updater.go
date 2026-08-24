// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package run

import "context"

// RunMetadataUpdateReason identifies a metadata update that active-run clients may observe.
// Add a reason here when a new metadata field needs an incremental client update.
type RunMetadataUpdateReason int

const (
	RunMetadataUpdateReasonUnknown RunMetadataUpdateReason = iota
	RunMetadataUpdateReasonSupportsStop
)

// MetadataUpdateNotifier observes persisted metadata changes for an active run.
type MetadataUpdateNotifier interface {
	OnRunMetadataChanged(RunMetadataUpdateReason)
}

// RunMetadataUpdater applies metadata mutations for an active run.
type RunMetadataUpdater struct {
	runID    RunID
	writer   *RunCollection
	notifier MetadataUpdateNotifier
}

func NewRunMetadataUpdater(runID RunID, writer *RunCollection, notifier MetadataUpdateNotifier) *RunMetadataUpdater {
	return &RunMetadataUpdater{runID: runID, writer: writer, notifier: notifier}
}

func (u *RunMetadataUpdater) AccumulateSupportsStop(ctx context.Context, supportsStop bool) error {
	unlock, err := u.writer.LockRun(ctx, u.runID)
	if err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			_ = unlock()
		}
	}()

	metadata, err := u.writer.readMetadata(u.runID)
	if err != nil {
		return err
	}
	if metadata.SupportsStop != nil {
		// Tool batches are discovered independently. A run supports Stop only if
		// every discovered tool does, so a prior false value is permanent.
		supportsStop = *metadata.SupportsStop && supportsStop
	}
	if metadata.SupportsStop != nil && *metadata.SupportsStop == supportsStop {
		return nil
	}
	metadata.SupportsStop = &supportsStop
	if err := u.writer.writeMetadataAtomic(u.runID, &metadata); err != nil {
		return err
	}
	unlockErr := unlock()
	locked = false
	if unlockErr != nil {
		return unlockErr
	}
	if u.notifier != nil {
		u.notifier.OnRunMetadataChanged(RunMetadataUpdateReasonSupportsStop)
	}
	return nil
}
