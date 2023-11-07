package app

import (
	"os"
	"os/signal"

	"github.com/olive-io/olive/meta"
	"github.com/olive-io/olive/pkg/signalutil"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/server/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewMetaCommand() *cobra.Command {
	app := &cobra.Command{
		Use:           "olive-meta",
		Short:         "a component of olive",
		Version:       version.GoV(),
		RunE:          runMeta,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	app.ResetFlags()
	flags := app.PersistentFlags()
	config.AddServerFlagSet(flags)
	config.AddLogFlagSet(flags)
	meta.AddFlagSet(flags)

	return app
}

func runMeta(cmd *cobra.Command, _ []string) error {
	flags := cmd.PersistentFlags()

	lcfg, err := config.LoggerConfigFromFlagSet(flags)
	if err != nil {
		return err
	}
	if err = lcfg.Apply(); err != nil {
		return err
	}

	logger := lcfg.GetLogger()

	mcfg, err := meta.ConfigFromFlagSet(flags)
	if err != nil {
		return err
	}
	if err = mcfg.Apply(); err != nil {
		return err
	}

	if err = setupMetaServer(logger, mcfg); err != nil {
		return err
	}

	return nil
}

func setupMetaServer(lg *zap.Logger, cfg meta.Config) error {
	ms, err := meta.NewServer(lg, cfg)
	if err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signalutil.Shutdown()...)

	if err = ms.Start(); err != nil {
		return err
	}

	select {
	// wait on kill signal
	case <-ch:
	}

	return ms.GracefulStop()
}
