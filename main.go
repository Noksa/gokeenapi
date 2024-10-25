package main

import (
	"context"
	"github.com/noksa/gokeenapi/cmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	ctx := context.Background()
	cmdCtx, _ := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	var err error
	errChan := make(chan error)
	go func() {
		errChan <- rootCmd.ExecuteContext(cmdCtx)
	}()
	select {
	case err = <-errChan:
	case <-cmdCtx.Done():
		err = cmdCtx.Err()
	}
	cmd.RestoreCursor()
	if err != nil {
		os.Exit(100)
	}
}
