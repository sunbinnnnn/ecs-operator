package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

func NewProvider(cloudName string) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, error) {
	clientOpts := new(clientconfig.ClientOpts)
	clientOpts.Cloud = cloudName
	opts, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, nil, err
	}
	opts.AllowReauth = true

	provider, err := openstack.NewClient(opts.IdentityEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("create providerClient err: %v", err)
	}

	err = openstack.Authenticate(provider, *opts)
	if err != nil {
		return nil, nil, fmt.Errorf("providerClient authentication err: %v", err)
	}
	return provider, clientOpts, nil
}
