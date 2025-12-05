package stackit

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/flatcar/mantle/platform"
	sdkconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	oapiError "github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"k8s.io/utils/ptr"
)

var (
	ErrorNotFound   = errors.New("not found")
	operatingSystem = "linux"
	osDistro        = "Flatcar"
	diskFormat      = "qcow2"
	DefaultLabels   = map[string]interface{}{
		"managed-by": "mantle",
	}
)

type API struct {
	client           *iaas.APIClient
	projectID        string
	region           string
	availabilityZone string
	machineType      string
	opts             *Options
}

type Options struct {
	*platform.Options
	ServiceAccountKeyPath string
	MachineType           string
	Region                string
	AvailabilityZone      string
	ProjectId             string
	ImageId               string
}

type Server struct {
	*iaas.Server
}

type Network struct {
	*iaas.Network
}

type Image struct {
	*iaas.Image
}

type Keypair struct {
	*iaas.Keypair
}

type SecurityGroup struct {
	*iaas.SecurityGroup
}

type PublicIP struct {
	*iaas.PublicIp
}

func New(opts *Options) (*API, error) {
	options := []sdkconfig.ConfigurationOption{
		sdkconfig.WithServiceAccountKeyPath(opts.ServiceAccountKeyPath),
	}
	client, err := iaas.NewAPIClient(options...)
	if err != nil {
		return nil, err
	}

	return &API{
		client:           client,
		projectID:        opts.ProjectId,
		region:           opts.Region,
		machineType:      opts.MachineType,
		availabilityZone: opts.AvailabilityZone,
		opts:             opts,
	}, nil
}

func (a *API) UploadImage(ctx context.Context, name, path, board string) (string, error) {
	var architecture string = "x86"
	if board == "arm64-usr" {
		architecture = "arm64"
	}
	imageConfig := iaas.ImageConfig{
		Architecture:          ptr.To(architecture),
		OperatingSystem:       ptr.To(operatingSystem),
		OperatingSystemDistro: createNullableString(osDistro),
	}

	imagePayload := iaas.CreateImagePayload{
		Config:     &imageConfig,
		DiskFormat: ptr.To(diskFormat),
		Name:       &name,
		Labels:     &DefaultLabels,
	}
	response, err := a.client.CreateImage(ctx, a.projectID, a.region).CreateImagePayload(imagePayload).Execute()
	if err != nil {
		return "", fmt.Errorf("creating image: %w", err)
	}
	log.Printf("Upload image to: %v", *response.UploadUrl)

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if inner := file.Close(); inner != nil {
			err = fmt.Errorf("failed to close file: %w, (%w)", inner, err)
		}
	}()
	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	url := response.GetUploadUrl()

	err = uploadFile(ctx, file, stat.Size(), url)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return response.GetId(), nil
}

func (a *API) CreateKeyPair(ctx context.Context, name, publicKey string) (*Keypair, error) {
	keypairPayload := iaas.CreateKeyPairPayload{
		PublicKey: ptr.To(publicKey),
		Name:      ptr.To(name),
		Labels:    &DefaultLabels,
	}
	keypairResponse, err := a.client.CreateKeyPair(ctx).CreateKeyPairPayload(keypairPayload).Execute()
	if err != nil {
		return nil, fmt.Errorf("error creating keypair: %s", err)
	}
	if isOpenAPINotFound(err) {
		return nil, ErrorNotFound
	}
	return &Keypair{keypairResponse}, err
}

func (a *API) DeleteKeyPair(ctx context.Context, name string) error {
	keypair, err := a.client.GetKeyPair(ctx, name).Execute()
	if err != nil {
		return fmt.Errorf("failed to get keypair: %w", err)
	}
	if keypair == nil {
		return nil
	}

	err = a.client.DeleteKeyPair(ctx, name).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete keypair: %w", err)
	}
	return nil
}

func (a *API) CreateServer(ctx context.Context, name iaas.CreateServerPayloadGetNameAttributeType, networkId iaas.CreateServerNetworkingGetNetworkIdAttributeType, securityGroups iaas.CreateServerPayloadGetSecurityGroupsAttributeType, keypairName iaas.CreateServerPayloadGetKeypairNameAttributeType, userData iaas.CreateServerPayloadGetUserDataAttributeType) (*Server, error) {
	networkingPayload := iaas.CreateServerPayloadGetNetworkingAttributeType(&iaas.CreateServerPayloadNetworking{
		CreateServerNetworking: &iaas.CreateServerNetworking{NetworkId: networkId},
	})

	bootVolumeSource := iaas.BootVolumeSource{
		Id:   ptr.To(a.opts.ImageId),
		Type: ptr.To("image"),
	}

	bootVolume := iaas.CreateServerPayloadGetBootVolumeAttributeType(&iaas.CreateServerPayloadBootVolume{
		DeleteOnTermination: ptr.To(true),
		PerformanceClass:    ptr.To("storage_premium_perf2"),
		Size:                ptr.To(int64(50)),
		Source:              &bootVolumeSource,
	})

	serverPayload := iaas.CreateServerPayload{
		AvailabilityZone: ptr.To(a.availabilityZone),
		BootVolume:       bootVolume,
		MachineType:      ptr.To(a.machineType),
		Name:             name,
		Networking:       networkingPayload,
		SecurityGroups:   securityGroups,
		UserData:         userData,
		Labels:           &DefaultLabels,
	}

	if keypairName != nil {
		serverPayload.KeypairName = keypairName
	}

	serverResponse, err := a.client.CreateServer(ctx, a.projectID, a.region).CreateServerPayload(serverPayload).Execute()

	if err != nil {
		return nil, fmt.Errorf("error creating server: %s", err)
	}
	server, err := wait.CreateServerWaitHandler(ctx, a.client, a.projectID, a.region, *serverResponse.Id).WaitWithContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("error creating server wait: %s", err)
	}
	return &Server{server}, nil
}

func (a *API) GetServer(ctx context.Context, id string) (*Server, error) {
	server, err := a.client.GetServer(ctx, a.projectID, a.region, id).Details(true).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	return &Server{server}, nil
}

func (a *API) DeleteServer(ctx context.Context, id string) error {
	server, err := a.client.GetServer(ctx, a.projectID, a.region, id).Execute()
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return nil
	}

	err = a.client.DeleteServer(ctx, a.projectID, a.region, id).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}
	return nil
}

func (a *API) CreateNetwork(ctx context.Context, name string) (*Network, error) {
	networkPayload := iaas.CreateNetworkPayload{
		Name:   ptr.To(name),
		Labels: &DefaultLabels,
	}
	networkResponse, err := a.client.CreateNetwork(ctx, a.projectID, a.region).CreateNetworkPayload(networkPayload).Execute()
	if isOpenAPINotFound(err) {
		return nil, ErrorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	network, err := wait.CreateNetworkWaitHandler(ctx, a.client, a.projectID, a.region, *networkResponse.Id).WaitWithContext(ctx)
	if isOpenAPINotFound(err) {
		return nil, ErrorNotFound
	}
	return &Network{network}, err
}

func (a *API) DeleteNetwork(ctx context.Context, id string) error {
	network, err := a.client.GetNetwork(ctx, a.projectID, a.region, id).Execute()
	if err != nil {
		return fmt.Errorf("failed to get network: %w", err)
	}
	if network == nil {
		return nil
	}

	err = a.client.DeleteNetwork(ctx, a.projectID, a.region, id).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}
	return nil
}

func (a *API) RemoveSecurityGroupFromServer(ctx context.Context, serverID string, securityGroupID string) error {
	err := a.client.RemoveSecurityGroupFromServer(ctx, a.projectID, a.region, serverID, securityGroupID).Execute()
	if err != nil {
		return fmt.Errorf("failed to remove security group from server: %w", err)
	}
	return nil
}

func (a *API) RemoveNetworkFromServer(ctx context.Context, serverID string, networkID string) error {
	err := a.client.RemoveNetworkFromServer(ctx, a.projectID, a.region, serverID, networkID).Execute()
	if err != nil {
		return fmt.Errorf("failed to remove server from network: %w", err)
	}
	return nil
}

func (a *API) GetSecurityGroup(ctx context.Context, securityGroupID string) (*SecurityGroup, error) {
	securityGroup, err := a.client.GetSecurityGroup(ctx, a.projectID, a.region, securityGroupID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get security group: %w", err)
	}
	return &SecurityGroup{securityGroup}, nil
}

func (a *API) CreateSecurityGroup(ctx context.Context, name string) (*SecurityGroup, error) {
	securityGroupPayload := iaas.CreateSecurityGroupPayload{
		Name:   ptr.To(name),
		Labels: &DefaultLabels,
	}
	securityGroup, err := a.client.CreateSecurityGroup(ctx, a.projectID, a.region).CreateSecurityGroupPayload(securityGroupPayload).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}
	return &SecurityGroup{securityGroup}, err
}

func (a *API) DeleteSecurityGroupRule(ctx context.Context, securityGroupID, securityGroupRuleID string) error {
	err := a.client.DeleteSecurityGroupRule(ctx, a.projectID, a.region, securityGroupID, securityGroupRuleID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete security group rule: %w", err)
	}
	return nil
}

func (a *API) DeleteSecurityGroup(ctx context.Context, securityGroupID string) error {
	securityGroup, err := a.client.GetSecurityGroup(ctx, a.projectID, a.region, securityGroupID).Execute()
	if err != nil {
		return fmt.Errorf("failed to get security group: %w", err)
	}
	if securityGroup == nil {
		return nil
	}

	err = a.client.DeleteSecurityGroup(ctx, a.projectID, a.region, securityGroupID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete security group: %w", err)
	}
	return nil
}

func (a *API) CreateSecurityGroupRuleTCP(ctx context.Context, securityGroupId string) error {
	protocol := iaas.StringAsCreateProtocol(ptr.To("tcp"))
	securityGroupRulePayload := iaas.CreateSecurityGroupRulePayload{
		Description: ptr.To("SSH access"),
		Direction:   ptr.To("ingress"),
		PortRange: &iaas.PortRange{
			Max: ptr.To(int64(65535)),
			Min: ptr.To(int64(1)),
		},
		IpRange:  ptr.To("0.0.0.0/0"),
		Protocol: &protocol,
	}
	_, err := a.client.CreateSecurityGroupRule(ctx, a.projectID, a.region, securityGroupId).CreateSecurityGroupRulePayload(securityGroupRulePayload).Execute()
	if err != nil {
		return fmt.Errorf("failed to create security group rule: %w", err)
	}
	return nil
}

func (a *API) CreateSecurityGroupRuleUDP(ctx context.Context, securityGroupId string) error {
	protocol := iaas.StringAsCreateProtocol(ptr.To("udp"))
	securityGroupRulePayload := iaas.CreateSecurityGroupRulePayload{
		Description: ptr.To("SSH access"),
		Direction:   ptr.To("ingress"),
		PortRange: &iaas.PortRange{
			Max: ptr.To(int64(65535)),
			Min: ptr.To(int64(1)),
		},
		IpRange:  ptr.To("0.0.0.0/0"),
		Protocol: &protocol,
	}
	_, err := a.client.CreateSecurityGroupRule(ctx, a.projectID, a.region, securityGroupId).CreateSecurityGroupRulePayload(securityGroupRulePayload).Execute()
	if err != nil {
		return fmt.Errorf("failed to create security group rule: %w", err)
	}
	return nil
}

func (a *API) CreateIPAddress(ctx context.Context) (*PublicIP, error) {
	ipPayload := iaas.CreatePublicIPPayload{
		Labels: &DefaultLabels,
	}
	ipAddress, err := a.client.CreatePublicIP(ctx, a.projectID, a.region).CreatePublicIPPayload(ipPayload).Execute()
	if err != nil {
		return nil, err
	}
	return &PublicIP{ipAddress}, nil
}

func (a *API) AttachPublicIPAddress(ctx context.Context, ipAddressId, serverId string) error {
	err := a.client.AddPublicIpToServer(ctx, a.projectID, a.region, serverId, ipAddressId).Execute()
	if err != nil {
		return fmt.Errorf("failed to add public ip to server: %w", err)
	}
	return nil
}

func (a *API) DeleteIPAddress(ctx context.Context, id string) error {
	ipAddress, err := a.client.GetPublicIP(ctx, a.projectID, a.region, id).Execute()
	if err != nil {
		return fmt.Errorf("failed to get public ip: %w", err)
	}
	if ipAddress == nil {
		return nil
	}

	err = a.client.DeletePublicIP(ctx, a.projectID, a.region, id).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete public ip: %w", err)
	}
	return nil
}

func (a *API) DeleteIPAddressByIP(ctx context.Context, ipAddress string) error {
	publicIPs, err := a.client.ListPublicIPs(ctx, a.projectID, a.region).Execute()
	if err != nil {
		return fmt.Errorf("failed to get public ips: %w", err)
	}
	for _, ip := range publicIPs.GetItems() {
		if *ip.Ip == ipAddress {
			err = a.client.DeletePublicIP(ctx, a.projectID, a.region, *ip.Id).Execute()
			if err != nil {
				return fmt.Errorf("failed to delete public ip: %w", err)
			}
		}
	}
	return nil
}

func isOpenAPINotFound(err error) bool {
	apiErr := &oapiError.GenericOpenAPIError{}
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound
}

func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error {
	createdCutoff := time.Now().Add(-gracePeriod)

	if err := a.gcServers(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc servers: %w", err)
	}

	if err := a.gcImages(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc images: %w", err)
	}

	if err := a.gcNetworks(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc networks: %w", err)
	}

	if err := a.gcSecurityGroups(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc security groups: %w", err)
	}

	if err := a.gcKeyPairs(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc keypairs: %w", err)
	}

	if err := a.gcPublicIPAddresses(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc public ip addresses: %w", err)
	}

	return nil
}

func uploadFile(ctx context.Context, reader io.Reader, filesize int64, url string) error {
	// code stolen from STACKIT CLI, they will provide this in the SDK in the near future
	// pass the file contents as stream, as they can get arbitrarily large. We do
	// _not_ want to load them into an internal buffer. The downside is, that we
	// have to set the content-length header manually
	uploadRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bufio.NewReader(reader))
	if err != nil {
		return fmt.Errorf("create image: cannot create request: %w", err)
	}
	uploadRequest.Header.Add("Content-Type", "application/octet-stream")
	uploadRequest.ContentLength = filesize

	uploadResponse, err := http.DefaultClient.Do(uploadRequest)
	if err != nil {
		return fmt.Errorf("create image: error contacting server for upload: %w", err)
	}
	defer func() {
		if inner := uploadResponse.Body.Close(); inner != nil {
			err = fmt.Errorf("error closing file: %w (%w)", inner, err)
		}
	}()
	if uploadResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("create image: server rejected image upload with %s", uploadResponse.Status)
	}

	return nil
}

func (a *API) gcImages(ctx context.Context, createdCutoff time.Time) error {
	response, err := a.client.ListImages(ctx, a.projectID, a.region).LabelSelector(labelSelector(DefaultLabels)).Execute()
	if err != nil {
		return fmt.Errorf("failed to list current images: %w", err)
	}

	for _, image := range response.GetItems() {
		if image.CreatedAt.After(createdCutoff) {
			continue
		}

		err := a.client.DeleteImage(ctx, a.projectID, a.region, *image.Id).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}

		_, err = wait.DeleteImageWaitHandler(ctx, a.client, a.projectID, a.region, *image.Id).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}
	}
	return nil
}

func (a *API) gcNetworks(ctx context.Context, createdCutoff time.Time) error {
	response, err := a.client.ListNetworks(ctx, a.projectID, a.region).LabelSelector(labelSelector(DefaultLabels)).Execute()
	if err != nil {
		return fmt.Errorf("failed to list current networks: %w", err)
	}

	for _, network := range response.GetItems() {
		if network.CreatedAt.After(createdCutoff) {
			continue
		}

		err := a.client.DeleteNetwork(ctx, a.projectID, a.region, *network.Id).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete network: %w", err)
		}

		_, err = wait.DeleteNetworkWaitHandler(ctx, a.client, a.projectID, a.region, *network.Id).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete network: %w", err)
		}

	}
	return nil
}

func (a *API) gcServers(ctx context.Context, createdCutoff time.Time) error {
	response, err := a.client.ListServers(ctx, a.projectID, a.region).LabelSelector(labelSelector(DefaultLabels)).Execute()
	if err != nil {
		return fmt.Errorf("failed to list current servers: %w", err)
	}

	for _, server := range response.GetItems() {
		if server.CreatedAt.After(createdCutoff) {
			continue
		}

		err := a.client.DeleteServer(ctx, a.projectID, a.region, *server.Id).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete server: %w", err)
		}

		_, err = wait.DeleteServerWaitHandler(ctx, a.client, a.projectID, a.region, *server.Id).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete server: %w", err)
		}
	}
	return nil
}

func (a *API) gcKeyPairs(ctx context.Context, createdCutoff time.Time) error {
	response, err := a.client.ListKeyPairs(ctx).LabelSelector(labelSelector(DefaultLabels)).Execute()
	if err != nil {
		return fmt.Errorf("failed to list current keys: %w", err)
	}

	for _, keyPair := range response.GetItems() {
		if keyPair.CreatedAt.After(createdCutoff) {
			continue
		}

		err := a.client.DeleteKeyPair(ctx, *keyPair.Name).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete keypair: %w", err)
		}
	}
	return nil
}

func (a *API) gcSecurityGroups(ctx context.Context, createdCutoff time.Time) error {
	response, err := a.client.ListSecurityGroups(ctx, a.projectID, a.region).LabelSelector(labelSelector(DefaultLabels)).Execute()
	if err != nil {
		return fmt.Errorf("failed to list current security groups: %w", err)
	}

	for _, group := range response.GetItems() {
		if group.CreatedAt.After(createdCutoff) {
			continue
		}

		err := a.client.DeleteSecurityGroup(ctx, a.projectID, a.region, *group.Id).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete security group: %w", err)
		}
	}
	return nil
}

func (a *API) gcPublicIPAddresses(ctx context.Context, createdCutoff time.Time) error {
	response, err := a.client.ListPublicIPs(ctx, a.projectID, a.region).LabelSelector(labelSelector(DefaultLabels)).Execute()
	if err != nil {
		return fmt.Errorf("failed to list current public IPs: %w", err)
	}

	for _, ip := range response.GetItems() {
		if ip.Labels == nil {
			return fmt.Errorf("no public IP labels found for %v", ip.Id)
		}

		createdAtValue, ok := (*ip.Labels)["createdAt"]
		if !ok {
			return fmt.Errorf("no createdAt label found for public IP %v", ip.Id)
		}

		dateStr, ok := createdAtValue.(string)
		if !ok {
			return fmt.Errorf("label 'createdAt' is not a string")
		}

		createdAtDate, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return fmt.Errorf("label 'createdAt' is not a RFC3339 date")
		}
		if createdAtDate.After(createdCutoff) {
			continue
		}

		err = a.client.DeletePublicIP(ctx, a.projectID, a.region, *ip.Id).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete public IP: %w", err)
		}
	}
	return nil
}

func labelSelector(labels map[string]interface{}) string {
	selectors := make([]string, 0, len(labels))

	for k, v := range labels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}

	// Reproducible result for tests
	sort.Strings(selectors)

	return strings.Join(selectors, ",")
}
