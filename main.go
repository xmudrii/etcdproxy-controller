package main

import (
	"fmt"

	"github.com/xmudrii/etcdproxy-controller/pkg/cmd/controller"
	"github.com/xmudrii/etcdproxy-controller/pkg/signals"
)

func main() {
	stopCh := signals.SetupSignalHandler()

	cmd := controller.NewCommandEtcdProxyControllerStart(stopCh)
	if err := cmd.Execute(); err != nil {
		fmt.Errorf("unable to run etcdproxy command: %v", err)
	}
}
