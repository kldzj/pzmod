package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/kldzj/pzmod/internal/cli"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/kldzj/pzmod/internal/version"
)

func main() {
	st, err := store.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pzmod:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	root := cli.NewRootCommand(st, version.Get())
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "pzmod:", err)
		os.Exit(1)
	}
}
