// Copyright 2018 Red Hat
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

package openstack

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/pkg/capnslog"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/pagination"
	ugroups "github.com/gophercloud/utils/v2/openstack/networking/v2/extensions/security/groups"

	"github.com/flatcar/mantle/auth"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/util"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/openstack")
)

type Options struct {
	*platform.Options

	// Config file. Defaults to $HOME/.config/openstack.json.
	ConfigPath string
	// Profile name
	Profile string

	// Region (e.g. "regionOne")
	Region string
	// Instance Flavor ID
	Flavor string
	// Image ID
	Image string
	// Network ID
	Network string
	// Domain ID
	Domain string
	// Floating IP Pool
	FloatingIPPool string
	// Host can be used to optionally SSH into deployed VMs from the OpenStack host
	Host string
	// User is the one used for the SSH connection to the Host
	User string
	// Keyfile is the abs. path to private SSH key file for the User on the Host
	Keyfile string
}

type Server struct {
	Server     *servers.Server
	FloatingIP *floatingips.FloatingIP
}

type API struct {
	opts *Options

	computeClient *gophercloud.ServiceClient
	imageClient   *gophercloud.ServiceClient
	networkClient *gophercloud.ServiceClient

	// floatingNetworkID is the UUID of the external network for floating IPs.
	// Resolved from Options.FloatingIPPool (Name or ID) during API init.
	floatingNetworkID string
}

func New(opts *Options) (*API, error) {
	ctx := context.Background()

	profiles, err := auth.ReadOpenStackConfig(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't read OpenStack config: %v", err)
	}

	if opts.Profile == "" {
		opts.Profile = "default"
	}
	profile, ok := profiles[opts.Profile]
	if !ok {
		return nil, fmt.Errorf("no such profile %q", opts.Profile)
	}

	if opts.Domain == "" {
		opts.Domain = profile.Domain
	}

	osOpts := gophercloud.AuthOptions{
		IdentityEndpoint: profile.AuthURL,
		TenantID:         profile.TenantID,
		TenantName:       profile.TenantName,
		Username:         profile.Username,
		Password:         profile.Password,
		DomainID:         profile.DomainID,
		// Enable automatic re‑authentication so that long‑running
		// kola test suites do not fail once the Keystone token
		// expires (typically after one hour).  With AllowReauth set
		// to true, gophercloud will transparently obtain a fresh
		// token whenever it receives a 401 response, preventing
		// intermittent "Authentication failed" errors during console
		// log retrieval, security‑group operations, etc.
		// See https://pkg.go.dev/github.com/gophercloud/gophercloud/v2#AuthOptions
		AllowReauth: true,
	}

	provider, err := openstack.AuthenticatedClient(ctx, osOpts)
	if err != nil {
		return nil, fmt.Errorf("failed creating provider: %v", err)
	}

	if opts.Region == "" {
		opts.Region = profile.Region
	}

	computeClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Name:   "nova",
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %v", err)
	}

	imageClient, err := openstack.NewImageV2(provider, gophercloud.EndpointOpts{
		Name:   "glance",
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create image client: %v", err)
	}

	networkClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network client: %v", err)
	}

	a := &API{
		opts:          opts,
		computeClient: computeClient,
		imageClient:   imageClient,
		networkClient: networkClient,
	}

	if a.opts.Flavor != "" {
		tmp, err := a.resolveFlavor(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving flavor: %v", err)
		}
		a.opts.Flavor = tmp
	}

	if a.opts.Image != "" {
		tmp, err := a.ResolveImage(a.opts.Image)
		if err != nil {
			return nil, fmt.Errorf("resolving image: %v", err)
		}
		a.opts.Image = tmp
	}

	if a.opts.Network != "" {
		tmp, err := a.resolveNetwork(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving network: %v", err)
		}
		a.opts.Network = tmp
	}

	if a.opts.FloatingIPPool == "" {
		a.opts.FloatingIPPool = profile.FloatingIPPool
	}

	if a.opts.FloatingIPPool != "" {
		netID, err := a.resolveFloatingNetwork(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving floating IP pool: %v", err)
		}
		a.floatingNetworkID = netID
	}

	return a, nil
}

func unwrapPages(ctx context.Context, pager pagination.Pager, allowEmpty bool) (pagination.Page, error) {
	if pager.Err != nil {
		return nil, fmt.Errorf("retrieving pager: %v", pager.Err)
	}

	pages, err := pager.AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving pages: %v", err)
	}

	if !allowEmpty {
		empty, err := pages.IsEmpty()
		if err != nil {
			return nil, fmt.Errorf("parsing pages: %v", err)
		}
		if empty {
			return nil, fmt.Errorf("empty pager")
		}
	}
	return pages, nil
}

func (a *API) resolveFlavor(ctx context.Context) (string, error) {
	pager := flavors.ListDetail(a.computeClient, flavors.ListOpts{})

	pages, err := unwrapPages(ctx, pager, false)
	if err != nil {
		return "", fmt.Errorf("flavors: %v", err)
	}

	flavors, err := flavors.ExtractFlavors(pages)
	if err != nil {
		return "", fmt.Errorf("extracting flavors: %v", err)
	}

	for _, flavor := range flavors {
		if flavor.ID == a.opts.Flavor || flavor.Name == a.opts.Flavor {
			return flavor.ID, nil
		}
	}

	return "", fmt.Errorf("specified flavor %q not found", a.opts.Flavor)
}

func (a *API) ResolveImage(img string) (string, error) {
	ctx := context.Background()
	// Use the Glance image service instead of Nova (removed in gophercloud v2).
	// Go through available images and match by ID or name.
	pager := images.List(a.imageClient, images.ListOpts{})

	pages, err := unwrapPages(ctx, pager, false)
	if err != nil {
		return "", fmt.Errorf("images: %v", err)
	}

	allImages, err := images.ExtractImages(pages)
	if err != nil {
		return "", fmt.Errorf("extracting images: %v", err)
	}

	for _, image := range allImages {
		if image.ID == img || image.Name == img {
			return image.ID, nil
		}
	}

	return "", fmt.Errorf("specified image %q not found", img)
}

func (a *API) resolveNetwork(ctx context.Context) (string, error) {
	nets, err := a.getNetworks(ctx)
	if err != nil {
		return "", err
	}

	for _, network := range nets {
		if network.ID == a.opts.Network || network.Name == a.opts.Network {
			return network.ID, nil
		}
	}

	return "", fmt.Errorf("specified network %q not found", a.opts.Network)
}

// resolveFloatingNetwork resolves FloatingIPPool to a network UUID.
// Pool may be name or UUID, so all networks listed and matched.
// If nothing matches, pool is treated as-is (raw UUID).
func (a *API) resolveFloatingNetwork(ctx context.Context) (string, error) {
	pager := networks.List(a.networkClient, networks.ListOpts{})

	pages, err := unwrapPages(ctx, pager, true)
	if err != nil {
		return "", fmt.Errorf("listing networks for floating IP pool: %v", err)
	}

	allNetworks, err := networks.ExtractNetworks(pages)
	if err != nil {
		return "", fmt.Errorf("extracting networks: %v", err)
	}

	for _, network := range allNetworks {
		if network.ID == a.opts.FloatingIPPool || network.Name == a.opts.FloatingIPPool {
			return network.ID, nil
		}
	}

	// If no match, assume FloatingIPPool as an external network UUID already.
	return a.opts.FloatingIPPool, nil
}

func (a *API) PreflightCheck() error {
	if err := servers.List(a.computeClient, servers.ListOpts{}).Err; err != nil {
		return fmt.Errorf("listing servers: %v", err)
	}
	return nil
}

func (a *API) CreateServer(name, sshKeyID, userdata string) (*Server, error) {
	ctx := context.Background()

	networkID := a.opts.Network
	if networkID == "" {
		networks, err := a.getNetworks(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting network: %v", err)
		}
		networkID = networks[0].ID
	}

	securityGroup, err := a.getSecurityGroup(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving security group: %v", err)
	}

	server, err := servers.Create(ctx, a.computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: servers.CreateOpts{
			Name:      name,
			FlavorRef: a.opts.Flavor,
			ImageRef:  a.opts.Image,
			Metadata: map[string]string{
				"CreatedBy": "mantle",
			},
			SecurityGroups: []string{securityGroup},
			Networks: []servers.Network{
				{
					UUID: networkID,
				},
			},
			UserData: []byte(userdata),
		},
		KeyName: sshKeyID,
	}, nil).Extract()
	if err != nil {
		return nil, fmt.Errorf("creating server: %v", err)
	}

	serverID := server.ID

	err = util.WaitUntilReady(5*time.Minute, 10*time.Second, func() (bool, error) {
		var err error
		server, err = servers.Get(ctx, a.computeClient, serverID).Extract()
		if err != nil {
			return false, err
		}
		return server.Status == "ACTIVE", nil
	})
	if err != nil {
		a.DeleteServer(serverID)
		return nil, fmt.Errorf("waiting for instance to run: %v", err)
	}

	var floatingip *floatingips.FloatingIP
	if a.opts.FloatingIPPool != "" {
		floatingip, err = a.createFloatingIP(ctx)
		if err != nil {
			a.DeleteServer(serverID)
			return nil, fmt.Errorf("creating floating ip: %v", err)
		}
		if err := a.associateFloatingIP(ctx, serverID, floatingip.ID); err != nil {
			a.DeleteServer(serverID)
			// Explicitly delete the floating ip as DeleteServer only deletes floating IPs that are
			// associated with servers
			a.deleteFloatingIP(ctx, floatingip.ID)
			return nil, fmt.Errorf("associating floating ip: %v", err)
		}

		server, err = servers.Get(ctx, a.computeClient, serverID).Extract()
		if err != nil {
			a.DeleteServer(serverID)
			return nil, fmt.Errorf("retrieving server info: %v", err)
		}
	}

	return &Server{
		Server:     server,
		FloatingIP: floatingip,
	}, nil
}

func (a *API) getNetworks(ctx context.Context) ([]networks.Network, error) {
	pager := networks.List(a.networkClient, networks.ListOpts{})

	pages, err := unwrapPages(ctx, pager, false)
	if err != nil {
		return nil, fmt.Errorf("networks: %v", err)
	}

	networks, err := networks.ExtractNetworks(pages)
	if err != nil {
		return nil, fmt.Errorf("extracting networks: %v", err)
	}
	return networks, nil
}

// getServerPorts returns the Neutron ports belonging to the given server.
// A VM's ports have their DeviceID set to the server UUID.
func (a *API) getServerPorts(ctx context.Context, serverID string) ([]ports.Port, error) {
	pager := ports.List(a.networkClient, ports.ListOpts{
		DeviceID: serverID,
	})

	pages, err := unwrapPages(ctx, pager, true)
	if err != nil {
		return nil, fmt.Errorf("listing ports for server %s: %v", serverID, err)
	}

	allPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, fmt.Errorf("extracting ports: %v", err)
	}
	return allPorts, nil
}

func (a *API) getSecurityGroup(ctx context.Context) (string, error) {
	id, err := ugroups.IDFromName(ctx, a.networkClient, "kola")
	if err != nil {
		if _, ok := err.(gophercloud.ErrResourceNotFound); ok {
			return a.createSecurityGroup(ctx)
		}
		return "", fmt.Errorf("finding security group: %v", err)
	}
	return id, nil
}

func (a *API) createSecurityGroup(ctx context.Context) (string, error) {
	securityGroup, err := groups.Create(ctx, a.networkClient, groups.CreateOpts{
		Name: "kola",
	}).Extract()
	if err != nil {
		return "", fmt.Errorf("creating security group: %v", err)
	}

	ruleSet := []struct {
		Direction      rules.RuleDirection
		EtherType      rules.RuleEtherType
		Protocol       rules.RuleProtocol
		PortRangeMin   int
		PortRangeMax   int
		RemoteGroupID  string
		RemoteIPPrefix string
	}{
		{
			Direction:     rules.DirIngress,
			EtherType:     rules.EtherType4,
			RemoteGroupID: securityGroup.ID,
		},
		{
			Direction:      rules.DirIngress,
			EtherType:      rules.EtherType4,
			Protocol:       rules.ProtocolTCP,
			PortRangeMin:   22,
			PortRangeMax:   22,
			RemoteIPPrefix: "0.0.0.0/0",
		},
		{
			Direction:     rules.DirIngress,
			EtherType:     rules.EtherType6,
			RemoteGroupID: securityGroup.ID,
		},
		{
			Direction:      rules.DirIngress,
			EtherType:      rules.EtherType4,
			Protocol:       rules.ProtocolTCP,
			PortRangeMin:   2379,
			PortRangeMax:   2380,
			RemoteIPPrefix: "0.0.0.0/0",
		},
	}

	for _, rule := range ruleSet {
		_, err = rules.Create(ctx, a.networkClient, rules.CreateOpts{
			Direction:      rule.Direction,
			EtherType:      rule.EtherType,
			SecGroupID:     securityGroup.ID,
			PortRangeMax:   rule.PortRangeMax,
			PortRangeMin:   rule.PortRangeMin,
			Protocol:       rule.Protocol,
			RemoteGroupID:  rule.RemoteGroupID,
			RemoteIPPrefix: rule.RemoteIPPrefix,
		}).Extract()
		if err != nil {
			a.deleteSecurityGroup(ctx, securityGroup.ID)
			return "", fmt.Errorf("adding security rule: %v", err)
		}
	}

	return securityGroup.ID, nil
}

func (a *API) deleteSecurityGroup(ctx context.Context, id string) error {
	return groups.Delete(ctx, a.networkClient, id).ExtractErr()
}

func (a *API) createFloatingIP(ctx context.Context) (*floatingips.FloatingIP, error) {
	return floatingips.Create(ctx, a.networkClient, floatingips.CreateOpts{
		FloatingNetworkID: a.floatingNetworkID,
	}).Extract()
}

// associateFloatingIP associates the given floating IP with the first port
// belonging to the server identified by serverID.
func (a *API) associateFloatingIP(ctx context.Context, serverID, floatingIPID string) error {
	serverPorts, err := a.getServerPorts(ctx, serverID)
	if err != nil {
		return fmt.Errorf("getting server ports: %v", err)
	}
	if len(serverPorts) == 0 {
		return fmt.Errorf("no port found for server %s", serverID)
	}

	portID := serverPorts[0].ID
	_, err = floatingips.Update(ctx, a.networkClient, floatingIPID, floatingips.UpdateOpts{
		PortID: &portID,
	}).Extract()
	return err
}

// disassociateFloatingIP detaches the floating IP from any port it is
// associated with.
func (a *API) disassociateFloatingIP(ctx context.Context, floatingIPID string) error {
	emptyPortID := ""
	_, err := floatingips.Update(ctx, a.networkClient, floatingIPID, floatingips.UpdateOpts{
		PortID: &emptyPortID,
	}).Extract()
	return err
}

func (a *API) deleteFloatingIP(ctx context.Context, id string) error {
	return floatingips.Delete(ctx, a.networkClient, id).ExtractErr()
}

// findFloatingIP returns the floating IP associated with the given server (if exists).
// A floating IP is considered associated with a server when its PortID matches
// one of the server's Neutron ports.
func (a *API) findFloatingIP(ctx context.Context, serverID string) (*floatingips.FloatingIP, error) {
	serverPorts, err := a.getServerPorts(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("getting server ports: %v", err)
	}
	portIDs := make(map[string]struct{}, len(serverPorts))
	for _, p := range serverPorts {
		portIDs[p.ID] = struct{}{}
	}

	pager := floatingips.List(a.networkClient, floatingips.ListOpts{})

	pages, err := unwrapPages(ctx, pager, true)
	if err != nil {
		return nil, fmt.Errorf("floating ips: %v", err)
	}

	allFIPs, err := floatingips.ExtractFloatingIPs(pages)
	if err != nil {
		return nil, fmt.Errorf("extracting floating ips: %v", err)
	}

	for i := range allFIPs {
		fip := allFIPs[i]
		if fip.PortID == "" {
			continue
		}
		if _, ok := portIDs[fip.PortID]; ok {
			return &fip, nil
		}
	}

	return nil, nil
}

// Deletes the server, and disassociates & deletes any floating IP associated with the given server.
func (a *API) DeleteServer(id string) error {
	ctx := context.Background()

	fip, err := a.findFloatingIP(ctx, id)
	if err != nil {
		return err
	}
	if fip != nil {
		if err := a.disassociateFloatingIP(ctx, fip.ID); err != nil {
			return fmt.Errorf("couldn't disassociate floating ip %s from server %s: %v", fip.ID, id, err)
		}
		if err := a.deleteFloatingIP(ctx, fip.ID); err != nil {
			// if the deletion of this floating IP fails then mantle cannot detect the floating IP was tied to the
			// server anymore. as such warn and continue deleting the server.
			plog.Warningf("couldn't delete floating ip %s: %v", fip.ID, err)
		}
	}

	if err := servers.Delete(ctx, a.computeClient, id).ExtractErr(); err != nil {
		return fmt.Errorf("deleting server: %v: %v", id, err)
	}

	return nil
}

func (a *API) GetConsoleOutput(id string) (string, error) {
	return servers.ShowConsoleOutput(context.Background(), a.computeClient, id, servers.ShowConsoleOutputOpts{}).Extract()
}

func (a *API) webUpload(ID, URI string) error {
	ctx := context.Background()
	createOpts := imageimport.CreateOpts{
		Name: imageimport.WebDownloadMethod,
		URI:  URI,
	}

	if err := imageimport.Create(ctx, a.imageClient, ID, createOpts).ExtractErr(); err != nil {
		return fmt.Errorf("importing web image: %w", err)
	}

	return nil
}

func (a *API) UploadImage(name, path string) (string, error) {
	ctx := context.Background()

	image, err := images.Create(ctx, a.imageClient, images.CreateOpts{
		Name:            name,
		ContainerFormat: "bare",
		DiskFormat:      "qcow2",
		Tags:            []string{"mantle"},
	}).Extract()
	if err != nil {
		return "", fmt.Errorf("creating image: %v", err)
	}

	u, err := url.Parse(path)
	if err == nil && u.Scheme != "" && image.ID != "" {
		plog.Debug("creating image from URL")
		if err := a.webUpload(image.ID, path); err != nil {
			a.DeleteImage(image.ID)
			return "", fmt.Errorf("web uploading: %w", err)
		}

		// It usually takes around 10 seconds to extract the image.
		if err := util.WaitUntilReady(1*time.Minute, 5*time.Second, func() (bool, error) {
			image, err = images.Get(ctx, a.imageClient, image.ID).Extract()
			if err != nil {
				return false, fmt.Errorf("getting image status: %w", err)
			}

			return image.Status == images.ImageStatusActive, nil
		}); err != nil {
			a.DeleteImage(image.ID)
			return "", fmt.Errorf("getting image active: %w", err)
		}

		return image.ID, nil
	}

	plog.Debug("creating image from source file")
	data, err := os.Open(path)
	if err != nil {
		a.DeleteImage(image.ID)
		return "", fmt.Errorf("opening image file: %v", err)
	}
	defer data.Close()

	err = imagedata.Upload(ctx, a.imageClient, image.ID, data).ExtractErr()
	if err != nil {
		a.DeleteImage(image.ID)
		return "", fmt.Errorf("uploading image data: %v", err)
	}

	return image.ID, nil
}

func (a *API) DeleteImage(imageID string) error {
	return images.Delete(context.Background(), a.imageClient, imageID).ExtractErr()
}

func (a *API) PruneKeys(olderThan time.Duration) error {
	ctx := context.Background()
	// Build a set of keypair names that are still in use by active servers so
	// that we don't delete keys that are currently required.
	usedKeys := make(map[string]struct{})

	srvPager := servers.List(a.computeClient, servers.ListOpts{})
	srvPages, err := unwrapPages(ctx, srvPager, true)
	if err != nil {
		return fmt.Errorf("listing servers: %v", err)
	}

	srvList, err := servers.ExtractServers(srvPages)
	if err != nil {
		return fmt.Errorf("extracting servers: %v", err)
	}

	for _, s := range srvList {
		if s.KeyName != "" {
			usedKeys[s.KeyName] = struct{}{}
		}
	}

	// List all keypairs in the project.
	kpPager := keypairs.List(a.computeClient, keypairs.ListOpts{})
	kpPages, err := unwrapPages(ctx, kpPager, true)
	if err != nil {
		return fmt.Errorf("listing keypairs: %v", err)
	}

	kpList, err := keypairs.ExtractKeyPairs(kpPages)
	if err != nil {
		return fmt.Errorf("extracting keypairs: %v", err)
	}

	now := time.Now()

	for _, kp := range kpList {
		// Skip keypairs that are still in use.
		if _, inUse := usedKeys[kp.Name]; inUse {
			continue
		}

		// Retrieve detailed information in order to obtain the optional
		// `created_at` field.
		var detail struct {
			Keypair struct {
				CreatedAt string `json:"created_at"`
			} `json:"keypair"`
		}

		if err := keypairs.Get(ctx, a.computeClient, kp.Name, nil).ExtractInto(&detail); err != nil {
			// If we fail to obtain details, skip deletion to be safe.
			plog.Warningf("could not get details for keypair %s: %v", kp.Name, err)
			continue
		}

		if detail.Keypair.CreatedAt == "" {
			// Missing creation timestamp – skip.
			continue
		}

		// Attempt to parse the timestamp using a small set of common layouts.
		var createdTime time.Time
		var parseErr error
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05.999999Z07:00", // micro-seconds + TZ
			"2006-01-02T15:04:05.999999",       // micro-seconds, no TZ
			"2006-01-02T15:04:05",              // seconds, no TZ
		} {
			createdTime, parseErr = time.Parse(layout, detail.Keypair.CreatedAt)
			if parseErr == nil {
				// If the layout did not specify timezone information (no "Z07"),
				// assume the timestamp is in UTC, which is what OpenStack typically
				// uses internally.
				if !strings.Contains(layout, "Z07") {
					createdTime = createdTime.UTC()
				}
				break
			}
		}
		if parseErr != nil {
			plog.Warningf("unable to parse created_at for keypair %s: %v", kp.Name, parseErr)
			continue
		}

		if now.Sub(createdTime) > olderThan {
			if err := a.DeleteKey(kp.Name); err != nil {
				plog.Warningf("failed deleting stale keypair %s: %v", kp.Name, err)
			} else {
				plog.Infof("deleted stale keypair %s (age %s)", kp.Name, now.Sub(createdTime))
			}
		} else {
			plog.Infof("skipping keypair %s (age %s)", kp.Name, now.Sub(createdTime))
		}
	}

	return nil
}

func (a *API) AddKey(name, key string) error {
	_, err := keypairs.Create(context.Background(), a.computeClient, keypairs.CreateOpts{
		Name:      name,
		PublicKey: key,
	}).Extract()
	return err
}

func (a *API) DeleteKey(name string) error {
	return keypairs.Delete(context.Background(), a.computeClient, name, nil).ExtractErr()
}

func (a *API) listServersWithMetadata(metadata map[string]string) ([]servers.Server, error) {
	ctx := context.Background()
	pager := servers.List(a.computeClient, servers.ListOpts{})

	pages, err := unwrapPages(ctx, pager, true)
	if err != nil {
		return nil, fmt.Errorf("servers: %v", err)
	}

	allServers, err := servers.ExtractServers(pages)
	if err != nil {
		return nil, fmt.Errorf("extracting servers: %v", err)
	}
	var retServers []servers.Server
	for _, server := range allServers {
		isMatch := true
		for key, val := range metadata {
			if value, ok := server.Metadata[key]; !ok || val != value {
				isMatch = false
				break
			}
		}
		if isMatch {
			retServers = append(retServers, server)
		}
	}
	return retServers, nil
}

func (a *API) listImagesWithTags(tags []string) ([]images.Image, error) {
	ctx := context.Background()
	listOpts := images.ListOpts{
		Tags: tags,
	}

	allPages, err := images.List(a.imageClient, listOpts).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing images: %w", err)
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		return nil, fmt.Errorf("extracting image list: %w", err)
	}

	return allImages, nil
}

func (a *API) GC(gracePeriod time.Duration) error {
	threshold := time.Now().Add(-gracePeriod)

	servers, err := a.listServersWithMetadata(map[string]string{
		"CreatedBy": "mantle",
	})
	if err != nil {
		return err
	}
	for _, server := range servers {
		if strings.Contains(server.Status, "DELETED") || server.Created.After(threshold) {
			continue
		}

		if err := a.DeleteServer(server.ID); err != nil {
			return fmt.Errorf("couldn't delete server %s: %v", server.ID, err)
		}
	}

	images, err := a.listImagesWithTags([]string{"mantle"})
	if err != nil {
		return fmt.Errorf("listing Mantle images: %w", err)
	}

	for _, image := range images {
		if image.CreatedAt.After(threshold) {
			continue
		}

		if err := a.DeleteImage(image.ID); err != nil {
			return fmt.Errorf("deleting image with name: %s", image.Name)
		}
	}

	err = a.PruneKeys(gracePeriod)
	if err != nil {
		return fmt.Errorf("pruning keys: %v", err)
	}

	return nil
}
