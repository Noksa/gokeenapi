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
	selected, _, findErr := rootCmd.Find(os.Args[1:])
	waitForShutdown := findErr == nil && (selected.Name() == cmd.CmdExporter || selected.Name() == cmd.CmdScheduler)
	var err error
	errChan := make(chan error, 1)
	go func() {
		errChan <- rootCmd.ExecuteContext(cmdCtx)
		close(errChan)
	}()
	select {
	case err = <-errChan:
	case <-cmdCtx.Done():
		// Services cooperate with cancellation and must drain their workers.
		// Other commands may be blocked on interactive input or legacy API calls.
		if waitForShutdown {
			err = <-errChan
		} else {
			err = cmdCtx.Err()
		}
	}
	cmd.RestoreCursor()
	if err != nil {
		gokeenlog.Info(color.HiRedString("Error occured!"))
		gokeenlog.Info(color.RedString(err.Error()))
		os.Exit(100)
	}
}
