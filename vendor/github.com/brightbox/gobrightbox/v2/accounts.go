package brightbox

import (
	"time"

	"github.com/brightbox/gobrightbox/v2/enums/accountstatus"
)

//go:generate ./generate_enum accountstatus pending active overdue warning suspended terminated closed deleted

// Account represents a Brightbox Cloud Account
// https://api.gb1.brightbox.com/1.0/#account
type Account struct {
	ResourceRef
	ID                    string
	Name                  string
	Status                accountstatus.Enum `json:"status"`
	Address1              string             `json:"address_1"`
	Address2              string             `json:"address_2"`
	City                  string
	County                string
	Postcode              string
	CountryCode           string     `json:"country_code"`
	CountryName           string     `json:"country_name"`
	VatRegistrationNumber string     `json:"vat_registration_number"`
	TelephoneNumber       string     `json:"telephone_number"`
	TelephoneVerified     bool       `json:"telephone_verified"`
	VerifiedTelephone     string     `json:"verified_telephone"`
	VerifiedIP            string     `json:"verified_ip"`
	ValidCreditCard       bool       `json:"valid_credit_card"`
	ServersUsed           uint       `json:"servers_used"`
	RAMLimit              uint       `json:"ram_limit"`
	RAMUsed               uint       `json:"ram_used"`
	DbsInstancesUsed      uint       `json:"dbs_instances_used"`
	DbsRAMLimit           uint       `json:"dbs_ram_limit"`
	DbsRAMUsed            uint       `json:"dbs_ram_used"`
	BlockStorageLimit     uint       `json:"block_storage_limit"`
	BlockStorageUsed      uint       `json:"block_storage_used"`
	CloudIPsLimit         uint       `json:"cloud_ips_limit"`
	CloudIPsUsed          uint       `json:"cloud_ips_used"`
	LoadBalancersLimit    uint       `json:"load_balancers_limit"`
	LoadBalancersUsed     uint       `json:"load_balancers_used"`
	LibraryFtpHost        string     `json:"library_ftp_host"`
	LibraryFtpUser        string     `json:"library_ftp_user"`
	LibraryFtpPassword    string     `json:"library_ftp_password"`
	CreatedAt             *time.Time `json:"created_at"`
	VerifiedAt            *time.Time `json:"verified_at"`
	Owner                 *User
	Clients               []APIClient
	Images                []Image
	Servers               []Server
	LoadBalancers         []LoadBalancer     `json:"load_balancers"`
	DatabaseServers       []DatabaseServer   `json:"database_servers"`
	DatabaseSnapshots     []DatabaseSnapshot `json:"database_snapshots"`
	CloudIPs              []CloudIP          `json:"cloud_ips"`
	ServerGroups          []ServerGroup      `json:"server_groups"`
	FirewallPolicies      []FirewallPolicy   `json:"firewall_policies"`
	Users                 []User
	Volumes               []Volume
	Zones                 []Zone
}

// AccountOptions is used to update objects
type AccountOptions struct {
	ID                    string  `json:"-"`
	Name                  *string `json:"name,omitempty"`
	Address1              *string `json:"address_1,omitempty"`
	Address2              *string `json:"address_2,omitempty"`
	City                  *string `json:"city,omitempty"`
	County                *string `json:"county,omitempty"`
	Postcode              *string `json:"postcode,omitempty"`
	CountryCode           *string `json:"country_code,omitempty"`
	VatRegistrationNumber *string `json:"vat_registration_number,omitempty"`
	TelephoneNumber       *string `json:"telephone_number,omitempty"`
}
