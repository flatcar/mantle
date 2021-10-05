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
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-11-01/network"

	"github.com/flatcar-linux/mantle/util"
)

type Machine struct {
	ID               string
	PublicIPAddress  string
	PrivateIPAddress string
	InterfaceName    string
	PublicIPName     string
}

func (a *API) getVMParameters(name, userdata, sshkey, storageAccountURI string, ip *network.PublicIPAddress, nic *network.Interface) compute.VirtualMachine {
	osProfile := compute.OSProfile{
		AdminUsername: util.StrToPtr("core"),
		ComputerName:  &name,
		LinuxConfiguration: &compute.LinuxConfiguration{
			SSH: &compute.SSHConfiguration{
				PublicKeys: &[]compute.SSHPublicKey{
					{
						Path:    util.StrToPtr("/home/core/.ssh/authorized_keys"),
						KeyData: &sshkey,
					},
				},
			},
		},
	}
	if userdata != "" {
		ud := base64.StdEncoding.EncodeToString([]byte(userdata))
		osProfile.CustomData = &ud
	}
	var imgRef *compute.ImageReference
	if a.opts.DiskURI != "" {
		imgRef = &compute.ImageReference{
			ID: &a.opts.DiskURI,
		}
	} else {
		imgRef = &compute.ImageReference{
			Publisher: &a.opts.Publisher,
			Offer:     &a.opts.Offer,
			Sku:       &a.opts.Sku,
			Version:   &a.opts.Version,
		}
	}
	return compute.VirtualMachine{
		Name:     &name,
		Location: &a.opts.Location,
		Tags: map[string]*string{
			"createdBy": util.StrToPtr("mantle"),
		},
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(a.opts.Size),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: imgRef,
				OsDisk: &compute.OSDisk{
					CreateOption: compute.DiskCreateOptionTypesFromImage,
				},
			},
			OsProfile: &osProfile,
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: nic.ID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: util.BoolToPtr(true),
						},
					},
				},
			},
			DiagnosticsProfile: &compute.DiagnosticsProfile{
				BootDiagnostics: &compute.BootDiagnostics{
					Enabled:    util.BoolToPtr(true),
					StorageURI: &storageAccountURI,
				},
			},
		},
	}
}

func (a *API) CreateInstance(name, userdata, sshkey, resourceGroup, storageAccount string) (*Machine, error) {
	subnet, err := a.getSubnet(resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("preparing network resources: %v", err)
	}

	ip, err := a.createPublicIP(resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("creating public ip: %v", err)
	}
	if ip.Name == nil {
		return nil, fmt.Errorf("couldn't get public IP name")
	}

	nic, err := a.createNIC(ip, &subnet, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("creating nic: %v", err)
	}
	if nic.Name == nil {
		return nil, fmt.Errorf("couldn't get NIC name")
	}

	vmParams := a.getVMParameters(name, userdata, sshkey, fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccount), ip, nic)
	plog.Infof("Creating Instance %s", name)

	future, err := a.compClient.CreateOrUpdate(context.TODO(), resourceGroup, name, vmParams)
	if err != nil {
		return nil, err
	}
	err = future.WaitForCompletionRef(context.TODO(), a.compClient.Client)
	if err != nil {
		return nil, err
	}
	_, err = future.Result(a.compClient)
	if err != nil {
		return nil, err
	}
	plog.Infof("Instance %s created", name)

	err = util.WaitUntilReady(5*time.Minute, 10*time.Second, func() (bool, error) {
		vm, err := a.compClient.Get(context.TODO(), resourceGroup, name, "")
		if err != nil {
			return false, err
		}

		if vm.VirtualMachineProperties.ProvisioningState != nil && *vm.VirtualMachineProperties.ProvisioningState != "Succeeded" {
			return false, nil
		}

		return true, nil
	})
	plog.Infof("Instance %s ready", name)
	if err != nil {
		_, _ = a.compClient.Delete(context.TODO(), resourceGroup, name, nil)
		_, _ = a.intClient.Delete(context.TODO(), resourceGroup, *nic.Name)
		_, _ = a.ipClient.Delete(context.TODO(), resourceGroup, *ip.Name)
		// TODO: remove disk which doesn't get removed automatically
		return nil, fmt.Errorf("waiting for machine to become active: %v", err)
	}

	vm, err := a.compClient.Get(context.TODO(), resourceGroup, name, "")
	if err != nil {
		return nil, err
	}

	if vm.Name == nil {
		return nil, fmt.Errorf("couldn't get VM ID")
	}

	publicaddr, privaddr, err := a.GetIPAddresses(*nic.Name, *ip.Name, resourceGroup)
	if err != nil {
		return nil, err
	}

	return &Machine{
		ID:               *vm.Name,
		PublicIPAddress:  publicaddr,
		PrivateIPAddress: privaddr,
		InterfaceName:    *nic.Name,
		PublicIPName:     *ip.Name,
	}, nil
}

// TerminateInstance deletes a VM created by CreateInstance with the public IP address and
// NIC created for it. Currently it does not delete the OS disk that is created (see TODO).
func (a *API) TerminateInstance(machine *Machine, resourceGroup string) error {
	future, err := a.compClient.Delete(context.TODO(), resourceGroup, machine.ID, nil)
	if err != nil {
		return err
	}
	err = future.WaitForCompletionRef(context.TODO(), a.compClient.Client)
	if err != nil {
		return err
	}
	_, err = future.Result(a.compClient)
	if err != nil {
		return err
	}

	ifFuture, err := a.intClient.Delete(context.TODO(), resourceGroup, machine.InterfaceName)
	if err != nil {
		return err
	}
	err = ifFuture.WaitForCompletionRef(context.TODO(), a.intClient.Client)
	if err != nil {
		return err
	}
	_, err = ifFuture.Result(a.intClient)
	if err != nil {
		return err
	}

	ipFuture, err := a.ipClient.Delete(context.TODO(), resourceGroup, machine.PublicIPName)
	if err != nil {
		return err
	}
	err = ipFuture.WaitForCompletionRef(context.TODO(), a.ipClient.Client)
	if err != nil {
		return err
	}
	_, err = ipFuture.Result(a.ipClient)
	if err != nil {
		return err
	}

	// TODO: remove disk which doesn't get removed automatically
	return nil
}

func (a *API) GetConsoleOutput(name, resourceGroup, storageAccount string) ([]byte, error) {
	kr, err := a.GetStorageServiceKeysARM(storageAccount, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("retrieving storage service keys: %v", err)
	}

	if kr.Keys == nil {
		return nil, fmt.Errorf("no storage service keys found")
	}
	k := *kr.Keys
	key := *k[0].Value

	vm, err := a.compClient.Get(context.TODO(), resourceGroup, name, compute.InstanceViewTypesInstanceView)
	if err != nil {
		return nil, fmt.Errorf("could not get VM: %v", err)
	}

	consoleURI := vm.VirtualMachineProperties.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI
	if consoleURI == nil {
		return nil, fmt.Errorf("serial console URI is nil")
	}

	// Only the full URI to the logs are present in the virtual machine
	// properties. Parse out the container & file name to use the GetBlob
	// API call directly.
	uri := []byte(*consoleURI)
	containerPat := regexp.MustCompile(`bootdiagnostics-[a-z0-9\-]+`)
	container := string(containerPat.Find(uri))
	if container == "" {
		return nil, fmt.Errorf("could not find container name in URI: %q", *consoleURI)
	}
	namePat := regexp.MustCompile(`[a-z0-9\-\.]+.serialconsole.log`)
	blobname := string(namePat.Find(uri))
	if blobname == "" {
		return nil, fmt.Errorf("could not find blob name in URI: %q", *consoleURI)
	}

	var data io.ReadCloser
	err = util.Retry(6, 10*time.Second, func() error {
		data, err = a.GetBlob(storageAccount, key, container, blobname)
		if err != nil {
			return fmt.Errorf("could not get blob for container %q, blobname %q: %v", container, blobname, err)
		}
		if data == nil {
			return fmt.Errorf("empty data while getting blob for container %q, blobname %q", container, blobname)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(data)
}
