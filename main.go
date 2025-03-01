/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/controllers"
	"github.com/milvus-io/milvus-operator/pkg/manager"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var workDir string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&workDir, "work-dir", "", "The work directory where the config assets locate")
	flag.StringVar(&controllers.ToolImage, "tool-image", controllers.ToolImage, "default tool image for setup milvus")
	flag.StringVar(&config.OperatorNamespace, "namespace", config.OperatorNamespace, "The namespace of self")
	flag.StringVar(&config.OperatorName, "name", config.OperatorName, "The name of self")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if err := config.Init(workDir); err != nil {
		setupLog.Error(err, "unable to init config")
		os.Exit(1)
	}

	mgr, err := manager.NewManager(metricsAddr, probeAddr, enableLeaderElection)
	if err != nil {
		setupLog.Error(err, "new manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	if err := controllers.SetupControllers(ctx, mgr, true); err != nil {
		setupLog.Error(err, "unable to setup controller with manager")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
