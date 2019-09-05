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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

type Service struct {
	provider       *gophercloud.ProviderClient
	computeClient  *gophercloud.ServiceClient
	identityClient *gophercloud.ServiceClient
	networkClient  *gophercloud.ServiceClient
	imagesClient   *gophercloud.ServiceClient
	volumeClient   *gophercloud.ServiceClient
}

func NewService(client *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts) (*Service, error) {
	identityClient, err := openstack.NewIdentityV3(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service client: %v", err)
	}

	computeClient, err := openstack.NewComputeV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service client: %v", err)
	}

	networkingClient, err := openstack.NewNetworkV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service client: %v", err)
	}

	imagesClient, err := openstack.NewImageServiceV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create image service client: %v", err)
	}

	volumeClient, err := openstack.NewBlockStorageV2(client, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create block service client: %v", err)
	}

	return &Service{
		provider:       client,
		identityClient: identityClient,
		computeClient:  computeClient,
		networkClient:  networkingClient,
		imagesClient:   imagesClient,
		volumeClient:   volumeClient,
	}, nil
}

// UpdateToken to update token if need.
func (svc *Service) UpdateToken() error {
	token := svc.provider.Token()
	result, err := tokens.Validate(svc.identityClient, token)
	if err != nil {
		return fmt.Errorf("validate token: %v", err)
	}
	if result {
		return nil
	}
	fmt.Print("Token is out of date, getting new token.")
	reAuthFunction := svc.provider.ReauthFunc
	if reAuthFunction() != nil {
		return fmt.Errorf("reAuth: %v", err)
	}
	return nil
}
