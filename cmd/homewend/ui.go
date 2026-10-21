// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ronalder100/homewend/internal/web"
)

// ui serves the interface until Ctrl-C, SIGTERM or until stdin closes: the
// desktop app holds our stdin, so if it dies without stopping us we stop
// anyway, and it stops us with SIGTERM when it quits.
func ui(args []string) int {
	newFlags("ui").Parse(args)
	web.Version = version
	srv, err := web.Listen()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	fmt.Println(srv.URL)
	srv.KeepJobs()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		io.Copy(io.Discard, os.Stdin)
		stop()
	}()
	served := make(chan error, 1)
	go func() { served <- srv.Serve() }()

	select {
	case err = <-served:
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = srv.Shutdown(shut)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	return exitOK
}
