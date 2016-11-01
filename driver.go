package main

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/state"
	"net"
	"time"
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

	// The Id of the target virtual LAN (VLAN).
	VLANID string

	// The name of the OS image used to create the machine.
	ImageName string

	// The Id of the OS image used to create the machine.
	ImageID string

	// The Id of the target (new) server.
	ServerID string

	// The initial password used to authenticate to target machines when installing the SSH key.
	SSHBootstrapPassword string

	// The CloudControl API client.
	client *compute.Client
}

// GetCreateFlags registers the "machine create" flags recognized by this driver, including
// their help text and defaults.
func (driver *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_USER",
			Name:   "cloudcontrol-user",
			Usage:  "The CloudControl user name",
			Value:  "",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_PASSWORD",
			Name:   "cloudcontrol-password",
			Usage:  "The CloudControl password",
			Value:  "",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_REGION",
			Name:   "cloudcontrol-region",
			Usage:  "The CloudControl region code",
			Value:  "",
		},
		mcnflag.StringFlag{
			Name:  "networkdomain",
			Usage: "The name of the target CloudControl network domain",
			Value: "",
		},
		mcnflag.StringFlag{
			Name:  "datacenter",
			Usage: "The Id of the data centre in which the the target CloudControl network domain is located",
			Value: "",
		},
		mcnflag.StringFlag{
			Name:  "vlan",
			Usage: "The Id of the target CloudControl VLAN",
			Value: "",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_SSH_USER",
			Name:   "ssh-user",
			Usage:  "The SSH username to use. Default: root",
			Value:  "root",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_SSH_KEY_FILE",
			Name:   "ssh-key-file",
			Usage:  "The SSH username to use. Default: root",
			Value:  "root",
		},
		mcnflag.StringFlag{
			EnvVar: "DD_COMPUTE_SSH_BOOTSTRAP_PASSWORD",
			Name:   "ssh-bootstrap-password",
			Usage:  "The initial SSH password used to bootstrap SSH key authentication.",
			Value:  "",
		},
	}
}

// DriverName returns the name of the driver
func (driver *Driver) DriverName() string {
	return "ddcloud"
}

// SetConfigFromFlags assigns and verifies the command-line arguments presented to the driver.
func (driver *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	driver.CloudControlUser = flags.String("cloudcontrol-user")
	driver.CloudControlPassword = flags.String("cloudcontrol-password")
	driver.CloudControlRegion = flags.String("cloudcontrol-region")

	driver.NetworkDomainName = flags.String("networkdomain")
	driver.DataCenterID = flags.String("datacenter")
	driver.VLANID = flags.String("vlan")
	driver.ImageName = DefaultImageName

	driver.SSHUser = flags.String("ssh-user")
	driver.SSHKeyPath = flags.String("ssh-key-file")
	driver.SSHBootstrapPassword = flags.String("ssh-bootstrap-password")

	return nil
}

// PreCreateCheck validates the configuration before making any changes.
func (driver *Driver) PreCreateCheck() error {
	log.Info("Examining target network domain (Id = '%s', region = '%s')...", driver.NetworkDomainID, driver.CloudControlRegion)

	err := driver.resolveNetworkDomain()
	if err != nil {
		return err
	}

	log.Info("Will create machine '%s' in network domain '%s' (data centre '%s').",
		driver.MachineName,
		driver.NetworkDomainName,
		driver.DataCenterID,
	)

	log.Info("Examining target VLAN (Id = '%s')...", driver.VLANID)
	vlan, err := driver.getVLAN()
	if err != nil {
		return err
	}
	if vlan == nil {
		log.Errorf("VLAN '%s' was not found in network domain '%s'.", driver.VLANID, driver.NetworkDomainID)

		return fmt.Errorf("VLAN '%s' was not found", driver.VLANID)
	}

	if vlan.NetworkDomain.ID != driver.NetworkDomainID {
		return fmt.Errorf("Cannot use VLAN '%s' because it belongs to a different network domain ('%s')", driver.VLANID, vlan.NetworkDomain.ID)
	}

	return driver.resolveOSImage()
}

// Create a new Docker Machine instance on CloudControl.
func (driver *Driver) Create() error {
	localPublicIP, err := getMyPublicIPv4Address()
	if err != nil {
		return err
	}
	log.Info("Local machine's public IP address is '%s'.", localPublicIP)

	log.Info("Deploying server '%s'...", driver.MachineName)
	server, err := driver.deployServer()
	if err != nil {
		return err
	}

	log.Info("Server '%s' has private IP '%s'.", driver.MachineName, driver.IPAddress)

	// TODO: Create NAT and firewall rules, if required.

	log.Info("Configuring SSH key for server '%s'...")
	err = driver.installSSHKey()
	if err != nil {
		return err
	}

	log.Info("Server '%s' has been successfully deployed.", server.Name)

	return nil
}

// GetState retrieves the status of a Docker Machine instance in CloudControl.
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
		log.Warn("Server '%s' not found; treating as already removed.", driver.ServerID)

		driver.ServerID = "" // Mark as deleted.

		return nil
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	err = client.DeleteServer(driver.ServerID)
	if err != nil {
		return err
	}

	err = client.WaitForDelete(compute.ResourceTypeServer, driver.ServerID, 10*time.Minute)
	if err != nil {
		return err
	}

	driver.ServerID = "" // Record deletion.

	return nil
}

// Start the target machine.
func (driver *Driver) Start() error {
	server, err := driver.getServer()
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("Server '%s' not found.", driver.ServerID)
	}

	if !server.Started {
		return nil
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	err = client.StartServer(driver.ServerID)
	if err != nil {
		return err
	}

	_, err = client.WaitForChange(compute.ResourceTypeServer, driver.ServerID, "Start server", 3*time.Minute)

	return err
}

// Stop the target machine (gracefully).
func (driver *Driver) Stop() error {
	server, err := driver.getServer()
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("Server '%s' not found.", driver.ServerID)
	}

	if !server.Started {
		return nil
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	err = client.ShutdownServer(driver.ServerID)
	if err != nil {
		return err
	}

	_, err = client.WaitForChange(compute.ResourceTypeServer, driver.ServerID, "Shut down server", 3*time.Minute)

	return err
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
	server, err := driver.getServer()
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("Server '%s' not found.", driver.ServerID)
	}

	if !server.Started {
		return nil
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	err = client.PowerOffServer(driver.ServerID)
	if err != nil {
		return err
	}

	_, err = client.WaitForChange(compute.ResourceTypeServer, driver.ServerID, "Power off server", 3*time.Minute)

	return err
}

// GetSSHHostname returns the hostname for SSH
func (driver *Driver) GetSSHHostname() (string, error) {
	return driver.IPAddress, nil
}

// GetSSHKeyPath returns the ssh key path
func (driver *Driver) GetSSHKeyPath() string {
	return driver.SSHKeyPath
}
