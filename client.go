package main

import (
	"errors"
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/docker/machine/libmachine/log"
	"time"
)

// Get the CloudControl API client used by the driver.
func (driver *Driver) getCloudControlClient() (client *compute.Client, err error) {
	client = driver.client
	if client != nil {
		return
	}

	if driver.CloudControlUser == "" {
		err = errors.New("Cannot connect to CloudControl API (user name has not been configured)")

		return
	}

	if driver.CloudControlPassword == "" {
		err = errors.New("Cannot connect to CloudControl API (password has not been configured)")

		return
	}

	if driver.CloudControlRegion == "" {
		err = errors.New("Cannot connect to CloudControl API (region not been configured)")

		return
	}

	driver.client = compute.NewClient(driver.CloudControlRegion, driver.CloudControlUser, driver.CloudControlPassword)

	return
}

// Determine whether the target server has been created.
func (driver *Driver) isServerCreated() bool {
	return driver.ServerID != ""
}

// Retrieve the target server (must have been created, or an error is returned).
func (driver *Driver) getServer() (*compute.Server, error) {
	if !driver.isServerCreated() {
		return nil, errors.New("Server has not been created")
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return nil, err
	}

	return client.GetServer(driver.ServerID)
}

// Retrieve the target network domain.
func (driver *Driver) getNetworkDomain() (*compute.NetworkDomain, error) {
	if driver.NetworkDomainID == "" {
		return nil, errors.New("Network domain Id has not been configured")
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return nil, err
	}

	return client.GetNetworkDomain(driver.ServerID)
}

// Resolve (find) the target network domain by name and data centre Id.
func (driver *Driver) resolveNetworkDomain() error {
	driver.NetworkDomainID = ""

	if driver.NetworkDomainName == "" {
		return errors.New("Network domain name has not been configured")
	}

	if driver.DataCenterID == "" {
		return errors.New("Data centre Id has not been configured")
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	networkDomain, err := client.GetNetworkDomainByName(driver.NetworkDomainName, driver.DataCenterID)
	if err != nil {
		return err
	}
	if networkDomain == nil {
		return fmt.Errorf("No network domain named '%s' was found in data centre '%s'", driver.NetworkDomainName, driver.DataCenterID)
	}

	driver.NetworkDomainID = networkDomain.ID

	return nil
}

// Retrieve the target VLAN.
func (driver *Driver) getVLAN() (*compute.VLAN, error) {
	if driver.VLANID == "" {
		return nil, errors.New("VLAN Id has not been configured")
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return nil, err
	}

	return client.GetVLAN(driver.VLANID)
}

// Resolve (find) the target OS image.
func (driver *Driver) resolveOSImage() error {
	driver.ImageID = ""

	networkDomain, err := driver.getNetworkDomain()
	if err != nil {
		return err
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	// Find target OS image.
	log.Info("Searching for image '%s' in data centre '%s'...", driver.ImageName, networkDomain.DatacenterID)

	image, err := client.FindOSImage(driver.ImageName, networkDomain.DatacenterID)
	if err == nil {
		return err
	}
	if image == nil {
		log.Errorf("OS image '%s' was not found in data centre '%s'.", driver.ImageName, networkDomain.DatacenterID)

		return fmt.Errorf("OS image '%s' was not found in data centre '%s'", driver.ImageName, networkDomain.DatacenterID)
	}

	driver.ImageID = image.ID

	return nil
}

// Retrieve the target OS image.
func (driver *Driver) getOSImage() (*compute.OSImage, error) {
	if driver.ImageID == "" {
		return nil, errors.New("Image Id has not been resolved")
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return nil, err
	}

	return client.GetOSImage(driver.ImageID)
}

func (driver *Driver) deployServer() (*compute.Server, error) {
	if driver.isServerCreated() {
		return nil, fmt.Errorf("Server '%s' already exists (Id = '%s')", driver.MachineName, driver.ServerID)
	}

	serverConfiguration, err := driver.buildDeploymentConfiguration()
	if err != nil {
		return nil, err
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return nil, err
	}

	driver.ServerID, err = client.DeployServer(serverConfiguration)
	if err != nil {
		return nil, err
	}

	log.Debug("Deploying server '%s' ('%s')...", driver.ServerID, driver.MachineName)

	resource, err := client.WaitForDeploy(compute.ResourceTypeServer, driver.ServerID, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	server := resource.(*compute.Server)

	log.Debug("Server '%s' ('%s') has been successfully provisioned...", driver.ServerID, server.Name)

	driver.IPAddress = *server.Network.PrimaryAdapter.PrivateIPv4Address

	return server, nil
}

// Build a deployment configuration for the target server.
func (driver *Driver) buildDeploymentConfiguration() (deploymentConfiguration compute.ServerDeploymentConfiguration, err error) {
	var image *compute.OSImage
	image, err = driver.getOSImage()
	if err != nil {
		return
	}
	if image == nil {
		err = fmt.Errorf("OS image '%s' not found", driver.ImageID)

		return
	}

	deploymentConfiguration = compute.ServerDeploymentConfiguration{
		Name:                  driver.MachineName,
		Description:           fmt.Sprintf("%s (created by Docker Machine).", driver.MachineName),
		AdministratorPassword: driver.SSHBootstrapPassword,

		Network: compute.VirtualMachineNetwork{
			NetworkDomainID: driver.NetworkDomainID,
			PrimaryAdapter: compute.VirtualMachineNetworkAdapter{
				VLANID: &driver.VLANID,
			},
		},
		PrimaryDNS:   "8.8.8.8",
		SecondaryDNS: "8.8.4.4",

		Start: true,
	}
	deploymentConfiguration.ApplyOSImage(image)

	return
}
