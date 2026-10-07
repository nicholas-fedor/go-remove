/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNotifyInterrupt_ReleaseIsSafeToRepeat verifies the signal handler can be
// released both from the watcher goroutine and from the caller.
//
// Releasing cancels the context, which wakes the watcher, which releases again,
// and main releases a third time. Two concurrent releases touching the same
// signal registration would race or panic, so this is run under the race
// detector. A second real signal is not delivered here because the default
// behavior restored after the first one would terminate this process. It
// registers process-wide signal handling, so it does not run in parallel.
func TestNotifyInterrupt_ReleaseIsSafeToRepeat(t *testing.T) {
	ctx, release := notifyInterrupt()

	require.NoError(t, ctx.Err())

	release()
	release()

	require.ErrorIs(t, ctx.Err(), context.Canceled)
}
