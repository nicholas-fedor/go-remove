/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package main is the go-remove process entry point.
//
// It owns signal handling and the only process exit. Everything else runs
// through app.Run.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/nicholas-fedor/go-remove/internal/app"
)

// main runs go-remove and exits with the code from app.Run.
//
// A cancellable context is installed for the interrupt and terminate signals,
// so a long move or copy can be stopped rather than killed part way through.
// The handler is released explicitly rather than deferred, so it is restored
// before os.Exit skips any deferred call.
func main() {
	ctx, stop := notifyInterrupt()

	code := app.Run(ctx)

	stop()
	os.Exit(code)
}

// notifyInterrupt installs signal handling where the first interrupt cancels the
// returned context and a second one terminates the process.
//
// Returns:
//   - context.Context: canceled by the first interrupt or terminate signal.
//   - context.CancelFunc: releases the handler. It is safe to call more than once.
func notifyInterrupt() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	// signal.NotifyContext leaves its handler registered after the first signal,
	// so a later signal would be swallowed rather than reaching the default
	// behavior. Releasing it once the context closes restores that behavior, and
	// also bounds the cancellation-free recovery of a half-finished deletion.
	go func() {
		<-ctx.Done()
		stop()
	}()

	return ctx, stop
}
