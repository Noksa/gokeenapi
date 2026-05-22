package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/cmd"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	ctx := context.Background()
	cmdCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
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
		gokeenlog.Info(color.HiRedString("Error occured!"))
		gokeenlog.Info(color.RedString(err.Error()))
		os.Exit(100)
	}
}
