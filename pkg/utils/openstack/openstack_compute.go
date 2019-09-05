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

package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type Instance struct {
	servers.Server
}

func (svc *Service) InstanceCreate(serverCreateOpts servers.CreateOpts) (*Instance, error) {
	server, err := servers.Create(svc.computeClient, serverCreateOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("create new server err: %v", err)
	}
	return &Instance{Server: *server}, nil
}

func (svc *Service) InstanceGet(instanceId string) (*Instance, error) {
	if instanceId == "" {
		return nil, fmt.Errorf("Instance ID should be specified to get detail info")
	}
	server, err := servers.Get(svc.computeClient, instanceId).Extract()
	if err != nil {
		return nil, fmt.Errorf("get instance %q detail failed: %v", instanceId, err)
	}
	return &Instance{Server: *server}, err
}

func (svc *Service) InstanceDelete(instanceId string) error {
	if instanceId == "" {
		return fmt.Errorf("Instance ID should be specified to delete")
	}
	return servers.Delete(svc.computeClient, instanceId).ExtractErr()
}

func (svc *Service) InstanceList() ([]*Instance, error) {
	listOpts := servers.ListOpts{}
	allPages, err := servers.List(svc.computeClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("get service list: %v", err)
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("extract services list: %v", err)
	}
	var instanceList []*Instance
	for _, server := range serverList {
		instanceList = append(instanceList, &Instance{
			Server: server,
		})
	}
	return instanceList, nil
}

func (svc *Service) InstanceForceDelete(instanceId string) error {
	if instanceId == "" {
		return fmt.Errorf("Instance ID should be specified to delete")
	}
	return servers.ForceDelete(svc.computeClient, instanceId).ExtractErr()
}