package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/xmudrii/etcdproxy-controller/pkg/cmd/controller"
	"github.com/xmudrii/etcdproxy-controller/pkg/signals"
	"k8s.io/apiserver/pkg/util/logs"

	// GCP authorization plugin needed to authorize in-cluster in GCP clusters.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	stopCh := signals.SetupSignalHandler()

	cmd := controller.NewCommandEtcdProxyControllerStart(stopCh)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		glog.Fatal(err)
	}
}
