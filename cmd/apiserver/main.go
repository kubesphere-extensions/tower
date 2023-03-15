package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v2"
	"k8s.io/klog/v2"

	"github.com/kubesphere-extensions/tower/pkg/apiserver"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:  "port",
				Usage: "HTTP server port",
				Value: 8080,
			},
			&cli.StringFlag{
				Name:  "kubeconfig",
				Usage: "kube config file path",
			},
		},
		Action: func(c *cli.Context) error {
			server, err := apiserver.New(c.Uint("port"), c.String("kubeconfig"))
			if err != nil {
				return err
			}
			if err = server.Run(context.TODO()); err != nil {
				return err
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		klog.Exit(err)
	}
}
