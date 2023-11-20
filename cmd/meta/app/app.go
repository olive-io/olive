package app

import (
	"os"
	"os/signal"

	"github.com/olive-io/olive/meta"
	"github.com/olive-io/olive/pkg/signalutil"
	"github.com/olive-io/olive/pkg/version"
	"github.com/spf13/cobra"
)

func NewMetaCommand() *cobra.Command {
	app := &cobra.Command{
		Use:           "olive-meta",
		Short:         "a component of olive",
		Version:       version.Version,
		RunE:          runMeta,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	app.ResetFlags()
	flags := app.PersistentFlags()
	meta.AddFlagSet(flags)

	return app
}

func runMeta(cmd *cobra.Command, _ []string) error {
	flags := cmd.PersistentFlags()

	cfg, err := meta.ConfigFromFlagSet(flags)
	if err != nil {
		return err
	}
	if err = cfg.Validate(); err != nil {
		return err
	}

	if err = setupMetaServer(cfg); err != nil {
		return err
	}

	return nil
}

func setupMetaServer(cfg meta.Config) error {
	ms, err := meta.NewServer(cfg)
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
