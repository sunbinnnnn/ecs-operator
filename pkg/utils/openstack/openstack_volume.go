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

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
)

type Volume struct {
	volumes.Volume
}

func (svc *Service) VolumeCreate(volumeCreateOpts volumes.CreateOpts) (*Volume, error) {
	volume, err := volumes.Create(svc.volumeClient, volumeCreateOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("create new volume err: %v", err)
	}
	return &Volume{Volume: *volume}, nil
}

func (svc *Service) VolumeGet(volumeId string) (*Volume, error) {
	if volumeId == "" {
		return nil, fmt.Errorf("volume ID should be specified to get detail info")
	}
	volume, err := volumes.Get(svc.volumeClient, volumeId).Extract()
	if err != nil {
		return nil, fmt.Errorf("get volume %q detail failed: %v", volumeId, err)
	}
	return &Volume{Volume: *volume}, err
}

func (svc *Service) VolumeDelete(volumeId string) error {
	if volumeId == "" {
		return fmt.Errorf("volume ID should be specified to delete")
	}
	deleteOpts := volumes.DeleteOpts{}
	return volumes.Delete(svc.volumeClient, volumeId, deleteOpts).ExtractErr()
}

func (svc *Service) VolumeList() ([]*Volume, error) {
	listOpts := volumes.ListOpts{}
	allPages, err := volumes.List(svc.volumeClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("get volume list failed: %v", err)
	}
	osVolumeList, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return nil, fmt.Errorf("extract volume list failed: %v", err)
	}
	var volumeList []*Volume
	for _, volume := range osVolumeList {
		volumeList = append(volumeList, &Volume{
			Volume: volume,
		})
	}
	return volumeList, nil
}
