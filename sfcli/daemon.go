package sfcli

import (
	"github.com/codegangsta/cli"
)

var (
	daemonCmd = cli.Command{
		Name:  "daemon",
		Usage: "daemon related commands",
		Subcommands: []cli.Command{
			daemonStartCmd,
		},
	}

	daemonStartCmd = cli.Command{
		Name:   "start",
		Usage:  "Start the SolidFire Docker Daemon: `start [options] NAME`",
		Action: cmdDaemonStart,
	}
)

func cmdDaemonStart(c *cli.Context) {
}
