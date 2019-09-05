/*
Copyright 2019 The Kubernetes Authors.

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
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/houming-wang/ecs-operator/pkg/controller"
	clientset "github.com/houming-wang/ecs-operator/pkg/generated/clientset/versioned"
	informers "github.com/houming-wang/ecs-operator/pkg/generated/informers/externalversions"
	"github.com/houming-wang/ecs-operator/pkg/signals"
	"github.com/houming-wang/ecs-operator/pkg/utils/openstack"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()
	flag.Set("alsologtostderr", "true")

	osProvider, clientOpts, err := openstack.NewProvider("ecsv5")
	if err != nil {
		glog.Fatalf("Error Creating OpenStack Provider: %s", err.Error())
	}
	osService, err := openstack.NewService(osProvider, clientOpts)
	if err != nil {
		glog.Fatalf("Error Creating OpenStack Services: %s", err.Error())
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	ecsClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building ecs clientset: %s", err.Error())
	}

	ecsInformerFactory := informers.NewSharedInformerFactory(ecsClient, time.Second*30)

	instanceController := controller.NewInstanceController(kubeClient, ecsClient, ecsInformerFactory, osService)

	go ecsInformerFactory.Start(stopCh)

	if err = instanceController.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}

	glog.Flush()
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
