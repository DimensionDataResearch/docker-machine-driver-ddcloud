package main

/*
 * Driver implementation
 * ---------------------
 */

import (
	"errors"
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/state"
	"net"
	"os"
)

// DefaultImageName is the name of the default OS image used to create machines.
const DefaultImageName = "Ubuntu 14.04 2 CPU"

// Driver is the Docker Machine driver for Dimension Data CloudControl.
type Driver struct {
	*drivers.BaseDriver

	// The CloudControl user name.
	CloudControlUser string

	// The CloudControl password
	CloudControlPassword string

	// The CloudControl region code
	CloudControlRegion string

	// The name of the target network domain.
	NetworkDomainName string

	// The Id of the data centre in which the target network domain is located.
	DataCenterID string

	// The Id of the target network domain.
	NetworkDomainID string

	// The name of the target virtual LAN (VLAN).
	VLANName string

	// The Id of the target virtual LAN (VLAN).
	VLANID string

	// The name of the OS image used to create the machine.
	ImageName string

	// The Id of the OS image used to create the machine.
	ImageID string

	// The Id of the target server.
	ServerID string

	// The private IPv4 address of the target server.
	PrivateIPAddress string

	// The Id of the NAT rule (if any) for the target server.
	NATRuleID string

	// The path to the SSH private key for the target server.
	SSHKey string

	// The initial password used to authenticate to target machines when installing the SSH key.
	SSHBootstrapPassword string

	// Create a firewall rule to allow SSH access to the taret server?
	CreateSSHFirewallRule bool

	// The Id of the firewall rule (if any) created for inbound SSH access to the target server.
	SSHFirewallRuleID string

	// The CloudControl API client.
	client *compute.Client
}

// GetCreateFlags registers the "machine create" flags recognized by this driver, including
// their help text and defaults.
func (driver *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_USER",
			Name:   "ddcloud-user",
			Usage:  "The CloudControl user name",
			Value:  "",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_PASSWORD",
			Name:   "ddcloud-password",
			Usage:  "The CloudControl password",
			Value:  "",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_REGION",
			Name:   "ddcloud-region",
			Usage:  "The CloudControl region name",
			Value:  "",
		},
		mcnflag.StringFlag{
			Name:  "ddcloud-networkdomain",
			Usage: "The name of the target CloudControl network domain",
			Value: "",
		},
		mcnflag.StringFlag{
			Name:  "ddcloud-datacenter",
			Usage: "The name of the data centre in which the the target CloudControl network domain is located",
			Value: "",
		},
		mcnflag.StringFlag{
			Name:  "ddcloud-vlan",
			Usage: "The name of the target CloudControl VLAN",
			Value: "",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_SSH_USER",
			Name:   "ddcloud-ssh-user",
			Usage:  "The SSH username to use. Default: root",
			Value:  "root",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_SSH_KEY",
			Name:   "ddcloud-ssh-key",
			Usage:  "The SSH key file to use",
			Value:  "",
		},
		mcnflag.IntFlag{
			EnvVar: "DD_COMPUTE_SSH_PORT",
			Name:   "ddcloud-ssh-port",
			Usage:  "The SSH port. Default: 22",
			Value:  22,
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_SSH_BOOTSTRAP_PASSWORD",
			Name:   "ddcloud-ssh-bootstrap-password",
			Usage:  "The initial SSH password used to bootstrap SSH key authentication",
			Value:  "",
		},
		mcnflag.BoolFlag{
			Name:  "ddcloud-create-ssh-firewall-rule",
			Usage: "Create a firewall rule to allow SSH access to the target server? Default: false",
		},
	}
}

// DriverName returns the name of the driver
func (driver *Driver) DriverName() string {
	return "ddcloud"
}

// SetConfigFromFlags assigns and verifies the command-line arguments presented to the driver.
func (driver *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	driver.CloudControlUser = flags.String("ddcloud-user")
	driver.CloudControlPassword = flags.String("ddcloud-password")
	driver.CloudControlRegion = flags.String("ddcloud-region")

	driver.NetworkDomainName = flags.String("ddcloud-networkdomain")
	driver.DataCenterID = flags.String("ddcloud-datacenter")
	driver.VLANName = flags.String("ddcloud-vlan")
	driver.ImageName = DefaultImageName

	driver.SSHPort = flags.Int("ddcloud-ssh-port")
	driver.SSHUser = flags.String("ddcloud-ssh-user")
	driver.SSHKey = flags.String("ddcloud-ssh-key")

	driver.SSHBootstrapPassword = flags.String("ddcloud-ssh-bootstrap-password")

	driver.CreateSSHFirewallRule = flags.Bool("ddcloud-create-ssh-firewall-rule")

	log.Debugf("docker-machine-driver-ddcloud %s", DriverVersion)

	return nil
}

// PreCreateCheck validates the configuration before making any changes.
func (driver *Driver) PreCreateCheck() error {
	log.Infof("Will create machine '%s' on VLAN '%s' in network domain '%s' (data centre '%s').",
		driver.MachineName,
		driver.VLANName,
		driver.NetworkDomainName,
		driver.DataCenterID,
	)

	log.Infof("Resolving target network domain '%s' in region '%s'...",
		driver.NetworkDomainName,
		driver.CloudControlRegion,
	)
	err := driver.resolveNetworkDomain()
	if err != nil {
		return err
	}

	log.Infof("Resolving target VLAN '%s' in network domain '%s'...",
		driver.VLANName,
		driver.NetworkDomainName,
	)
	err = driver.resolveVLAN()
	if err != nil {
		return err
	}

	log.Infof("Resolving OS image '%s' in data centre '%s'...",
		driver.ImageName,
		driver.DataCenterID,
	)
	err = driver.resolveOSImage()
	if err != nil {
		return err
	}

	return nil
}

// Create a new Docker Machine instance on CloudControl.
func (driver *Driver) Create() error {
	log.Infof("Importing SSH key...")

	err := driver.importSSHKey()
	if err != nil {
		return err
	}

	log.Infof("Creating server '%s'...", driver.MachineName)
	server, err := driver.deployServer()
	if err != nil {
		return err
	}

	log.Infof("Exposing server '%s'...", driver.MachineName)
	err = driver.createNATRuleForServer()
	if err != nil {
		return err
	}

	log.Infof("Server '%s' has private IP '%s'.", driver.MachineName, driver.PrivateIPAddress)
	log.Infof("Server '%s' has public IP '%s'.", driver.MachineName, driver.IPAddress)

	if driver.CreateSSHFirewallRule {
		var clientPublicIPAddress string
		clientPublicIPAddress, err = getClientPublicIPv4Address()
		if err != nil {
			return err
		}

		log.Infof("Creating firewall rule to enable inbound SSH traffic from local machine '%s' ('%s') to '%s' ('%s':%d)...",
			os.Getenv("HOST"),
			clientPublicIPAddress,
			driver.MachineName,
			driver.IPAddress,
			driver.SSHPort,
		)

		err = driver.createSSHFirewallRule(clientPublicIPAddress)
		if err != nil {
			return err
		}
	}

	log.Infof("Installing SSH key for server '%s' ('%s')...", driver.MachineName, driver.IPAddress)
	err = driver.installSSHKey()
	if err != nil {
		return err
	}

	log.Infof("Server '%s' has been successfully created.", server.Name)

	return nil
}

// GetState retrieves the status of the target Docker Machine instance in CloudControl.
func (driver *Driver) GetState() (state.State, error) {
	server, err := driver.getServer()
	if err != nil {
		return state.None, err
	}
	if server == nil {
		return state.None, nil // Server does not exist.
	}

	if !server.Deployed {
		return state.Starting, nil // Server is being deployed
	}

	if server.Started {
		return state.Running, nil // Server is running
	}

	return state.Stopped, nil // Server is stopped.
}

// GetURL returns docker daemon URL on the target machine
func (driver *Driver) GetURL() (string, error) {
	if driver.IPAddress == "" {
		return "", nil
	}

	url := fmt.Sprintf("tcp://%s", net.JoinHostPort(driver.IPAddress, "2376"))

	return url, nil
}

// Remove deletes the target machine.
func (driver *Driver) Remove() error {
	server, err := driver.getServer()
	if err != nil {
		return err
	}
	if server == nil {
		log.Warnf("Server '%s' not found; treating as already removed.", driver.ServerID)

		driver.ServerID = "" // Mark as deleted.

		return nil
	}

	if server.Started {
		err = driver.Stop()
		if err != nil {
			return err
		}
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	if driver.isSSHFirewallRuleCreated() {
		err = driver.deleteSSHFirewallRule()
		if err != nil {
			return err
		}
	}

	if driver.isNATRuleCreated() {
		err = driver.deleteNATRuleForServer()
		if err != nil {
			return err
		}
	}

	err = client.DeleteServer(driver.ServerID)
	if err != nil {
		return err
	}

	err = client.WaitForDelete(compute.ResourceTypeServer, driver.ServerID, serverDeleteTimeout)
	if err != nil {
		return err
	}

	driver.ServerID = "" // Record deletion.

	return nil
}

// Start the target machine.
func (driver *Driver) Start() error {
	return driver.startServer()
}

// Stop the target machine (gracefully).
func (driver *Driver) Stop() error {
	return driver.stopServer()
}

// Restart the target machine.
func (driver *Driver) Restart() error {
	err := driver.Stop()
	if err != nil {
		return err
	}

	return driver.Start()
}

// Kill the target machine (hard shutdown).
func (driver *Driver) Kill() error {
	return driver.powerOffServer()
}

// GetSSHHostname returns the hostname for SSH
func (driver *Driver) GetSSHHostname() (string, error) {
	if !driver.isServerCreated() {
		return "", errors.New("Server has not been created")
	}

	return driver.IPAddress, nil
}

// GetSSHKeyPath returns the ssh key path
func (driver *Driver) GetSSHKeyPath() string {
	return driver.SSHKeyPath
}
