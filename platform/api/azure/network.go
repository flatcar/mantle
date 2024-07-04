// Copyright 2018 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
)

var (
	singleVirtualNetworkPrefix = "10.0.0.0/16"
	virtualNetworkPrefix       = []*string{&singleVirtualNetworkPrefix}
	subnetPrefix               = "10.0.0.0/24"
	kolaSubnet                 = "kola-subnet"
	kolaVnet                   = "kola-vn"
)

func (a *API) findVnetSubnet(vnetSubnetStr string) (Network, error) {
	parts := strings.SplitN(vnetSubnetStr, "/", 2)
	vnetName := parts[0]
	subnetName := "default"
	if len(parts) > 1 {
		subnetName = parts[1]
	}
	var net *armnetwork.VirtualNetwork
	pager := a.netClient.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(context.TODO())
		if err != nil {
			return Network{}, fmt.Errorf("failed to iterate vnets: %w", err)
		}
		for _, vnet := range page.Value {
			if vnet.Name != nil && *vnet.Name == vnetName {
				net = vnet
				break
			}
		}
		if net != nil {
			break
		}
	}
	if net == nil {
		return Network{}, fmt.Errorf("failed to find vnet %s", vnetName)
	}
	subnets := net.Properties.Subnets
	if subnets == nil {
		return Network{}, fmt.Errorf("failed to find subnet %s in vnet %s", subnetName, vnetName)
	}
	for _, subnet := range subnets {
		if subnet != nil && subnet.Name != nil && *subnet.Name == subnetName {
			return Network{*subnet}, nil
		}
	}
	return Network{}, fmt.Errorf("failed to find subnet %s in vnet %s", subnetName, vnetName)
}

func (a *API) PrepareNetworkResources(resourceGroup string) (Network, error) {
	if a.Opts.VnetSubnetName != "" {
		return a.findVnetSubnet(a.Opts.VnetSubnetName)
	}

	if err := a.createVirtualNetwork(resourceGroup); err != nil {
		return Network{}, err
	}

	subnet, err := a.createSubnet(resourceGroup)
	if err != nil {
		return Network{}, err
	}
	return Network{subnet}, nil
}

func (a *API) createVirtualNetwork(resourceGroup string) error {
	plog.Infof("Creating VirtualNetwork %s", kolaVnet)
	poller, err := a.netClient.BeginCreateOrUpdate(context.TODO(), resourceGroup, kolaVnet, armnetwork.VirtualNetwork{
		Location: &a.Opts.Location,
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: virtualNetworkPrefix,
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(context.TODO(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (a *API) createSubnet(resourceGroup string) (armnetwork.Subnet, error) {
	plog.Infof("Creating Subnet %s", kolaSubnet)
	poller, err := a.subClient.BeginCreateOrUpdate(context.TODO(), resourceGroup, kolaVnet, kolaSubnet, armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: &subnetPrefix,
		},
	}, nil)
	if err != nil {
		return armnetwork.Subnet{}, err
	}
	r, err := poller.PollUntilDone(context.TODO(), nil)
	if err != nil {
		return armnetwork.Subnet{}, err
	}
	return r.Subnet, nil
}

func (a *API) getSubnet(resourceGroup, vnet, subnet string) (armnetwork.Subnet, error) {
	r, err := a.subClient.Get(context.TODO(), resourceGroup, vnet, subnet, nil)
	return r.Subnet, err
}

func (a *API) createPublicIP(resourceGroup string) (*armnetwork.PublicIPAddress, error) {
	name := randomName("ip")
	plog.Infof("Creating PublicIP %s", name)

	poller, err := a.ipClient.BeginCreateOrUpdate(context.TODO(), resourceGroup, name, armnetwork.PublicIPAddress{
		Location: &a.Opts.Location,
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			DeleteOption: to.Ptr(armnetwork.DeleteOptionsDelete),
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	r, err := poller.PollUntilDone(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	ip := r.PublicIPAddress
	ip.Properties.DeleteOption = to.Ptr(armnetwork.DeleteOptionsDelete)
	return &ip, nil
}

func (a *API) getPublicIP(name, resourceGroup string) (string, error) {
	ip, err := a.ipClient.Get(context.TODO(), resourceGroup, name, nil)
	if err != nil {
		return "", err
	}

	if ip.Properties.IPAddress == nil {
		return "", fmt.Errorf("IP Address is nil")
	}

	return *ip.Properties.IPAddress, nil
}

// returns PublicIP, PrivateIP, error
func (a *API) GetIPAddresses(name, publicIPName, resourceGroup string) (string, string, error) {
	privateIP, err := a.GetPrivateIP(name, resourceGroup)
	if err != nil {
		return "", "", err
	}
	if publicIPName == "" {
		return privateIP, privateIP, nil
	}

	publicIP, err := a.getPublicIP(publicIPName, resourceGroup)
	if err != nil {
		return "", "", err
	}
	return publicIP, privateIP, nil
}

func (a *API) GetPrivateIP(name, resourceGroup string) (string, error) {
	nic, err := a.intClient.Get(context.TODO(), resourceGroup, name, nil)
	if err != nil {
		return "", err
	}
	var privateIP *string
	for _, conf := range nic.Properties.IPConfigurations {
		if conf == nil || conf.Properties == nil || conf.Properties.PrivateIPAddress == nil {
			//return "", "", fmt.Errorf("PrivateIPAddress is nil")
			continue
		}
		privateIP = conf.Properties.PrivateIPAddress
		break
	}
	if privateIP == nil {
		return "", fmt.Errorf("no ip configurations found")
	}
	return *privateIP, nil
}

func (a *API) createNIC(ip *armnetwork.PublicIPAddress, subnet *armnetwork.Subnet, resourceGroup string) (*armnetwork.Interface, error) {
	name := randomName("nic")
	ipconf := randomName("nic-ipconf")
	plog.Infof("Creating NIC %s", name)

	poller, err := a.intClient.BeginCreateOrUpdate(context.TODO(), resourceGroup, name, armnetwork.Interface{
		Location: &a.Opts.Location,
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: &ipconf,
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						PublicIPAddress:           ip,
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
						Subnet:                    subnet,
					},
				},
			},
			EnableAcceleratedNetworking: to.Ptr(true),
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	r, err := poller.PollUntilDone(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	return &r.Interface, nil
}
