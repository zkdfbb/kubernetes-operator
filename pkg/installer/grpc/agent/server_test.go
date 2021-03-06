/*
 * Copyright 2019 gosoon.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"net"
	"testing"

	"github.com/gosoon/glog"
	ecsv1 "github.com/gosoon/kubernetes-operator/pkg/apis/ecs/v1"
	installerv1 "github.com/gosoon/kubernetes-operator/pkg/apis/installer/v1"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const port = "10023"

func TestInstallCluster(t *testing.T) {
	// start grpc server
	l, err := net.Listen("tcp", ":"+"10023")
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	agent := NewAgent(&Options{
		Server: server,
		Port:   port,
	})

	// register grpc server
	installerv1.RegisterInstallerServer(server, agent)

	go func() {
		glog.Fatal(server.Serve(l))
	}()

	kubernetesCluster := &ecsv1.KubernetesCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesCluster",
			APIVersion: "ecs.yun.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: ecsv1.KubernetesClusterSpec{
			Cluster: ecsv1.Cluster{
				ClusterType:          ecsv1.KubernetesClusterType,
				PodCIDR:              "192.168.0.0/16",
				ServiceCIDR:          "10.233.0.0/18",
				MasterList:           []ecsv1.Node{{IP: "192.168.72.224", Role: ecsv1.ControlPlaneRole}},
				ExternalLoadBalancer: "127.0.0.1",
				Region:               "default",
				KubeVersion:          "v1.15.3",
			},
		},
	}

	err = agent.ClusterNew(kubernetesCluster)
	if err != nil {
		glog.Error(err)
	}

}
