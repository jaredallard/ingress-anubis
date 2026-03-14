// Copyright (C) 2026 ingress-anubis contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: GPL-3.0

// Package main implements ingress-anubis.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/jaredallard/ingress-anubis/internal/config"
	"github.com/jaredallard/ingress-anubis/internal/controller"
	"go.rgst.io/jaredallard/slogext/v2"
)

func entrypoint(log slogext.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	svc := controller.NewKubernetesService(cfg, log)
	return svc.Run(ctx)
}

func main() {
	log := slogext.New()
	if err := entrypoint(log); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run: %v\n", err)
		os.Exit(1)
	}
}
