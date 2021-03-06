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

package create

import (
	"fmt"
	"os"

	"github.com/gosoon/glog"
	"github.com/gosoon/kubernetes-operator/pkg/installer/cluster/create"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/context"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/actions"
	configaction "github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/actions/config"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/actions/installcni"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/actions/kubeadminit"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/actions/kubeadmjoin"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/actions/waitforready"
	createtypes "github.com/gosoon/kubernetes-operator/pkg/internal/cluster/create/types"
	"github.com/gosoon/kubernetes-operator/pkg/internal/cluster/delete"
	"github.com/gosoon/kubernetes-operator/pkg/internal/util/cli"
)

// Cluster creates a cluster
func Cluster(ctx *context.Context, options ...create.ClusterOption) error {
	// apply options, do defaulting etc.
	opts, err := collectOptions(options...)
	if err != nil {
		return err
	}

	fmt.Printf("clusterOptions:%+v \n", opts)

	if err := validate(opts); err != nil {
		return err
	}

	ctx.ClusterOptions = opts

	status := cli.NewStatus(os.Stdout)
	// pull docker image
	// attempt to explicitly pull the required node images if they doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	ensureNodeImages(status, opts.NodeImage)

	// prepare node, copy docker image bin to local path and  为 node 和 master 生成配置文件
	// Create node containers implementing defined config Nodes
	if err := provisionNodes(status, ctx); err != nil {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		glog.Error(err)

		// if exec failed and cleanup
		_ = delete.Cluster(ctx)
		return err
	}

	// this step is setup kubeadm config
	actionsToRun := []actions.Action{
		configaction.NewAction(),
	}

	if opts.SetupKubernetes {
		// start control plane
		// run kubeadm init
		actionsToRun = append(actionsToRun,
			kubeadminit.NewAction(),
		)

		// this step might be skipped, but is next after init
		// this step is installing cni,default networking is calico
		if !opts.Config.Networking.DisableDefaultCNI {
			actionsToRun = append(actionsToRun,
				installcni.NewAction(),
			)
		}

		// add remaining steps
		// if current is worker node, run kubeadm join
		// and wait for cluster readiness
		actionsToRun = append(actionsToRun,
			kubeadmjoin.NewAction(),
			waitforready.NewAction(opts.WaitForReady),
		)
	}

	actionsContext := actions.NewActionContext(opts, ctx.Server, ctx.Port, status)
	for _, action := range actionsToRun {
		if err := action.Execute(actionsContext); err != nil {
			// if failed and cleanup
			_ = delete.Cluster(ctx)
			return err
		}
	}

	return nil

}

func collectOptions(options ...create.ClusterOption) (*createtypes.ClusterOptions, error) {
	// apply options
	opts := &createtypes.ClusterOptions{
		SetupKubernetes: true,
	}
	for _, option := range options {
		newOpts, err := option(opts)
		if err != nil {
			return nil, err
		}
		opts = newOpts
	}

	return opts, nil
}
