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

	client = compute.NewClient(driver.CloudControlRegion, driver.CloudControlUser, driver.CloudControlPassword)
	client.ConfigureRetry(10, 5*time.Second)

	driver.client = client

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

// Resolve (find) the target network domain by name and data centre Id.
func (driver *Driver) resolveVLAN() error {
	driver.VLANID = ""

	if driver.VLANName == "" {
		return errors.New("VLAN name has not been configured")
	}

	var err error
	if driver.NetworkDomainID == "" {
		err = driver.resolveNetworkDomain()
		if err != nil {
			return err
		}
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	vlan, err := client.GetVLANByName(driver.VLANName, driver.NetworkDomainID)
	if err != nil {
		return err
	}
	if vlan == nil {
		return fmt.Errorf("No VLAN named '%s' was found in network domain '%s' ('%s')", driver.VLANName, driver.NetworkDomainName, driver.NetworkDomainID)
	}

	driver.VLANID = vlan.ID

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

	image, err := client.FindOSImage(driver.ImageName, driver.DataCenterID)
	if err != nil {
		return err
	}
	if image == nil {
		log.Errorf("OS image '%s' was not found in data centre '%s'.", driver.ImageName, networkDomain.DatacenterID)

		return fmt.Errorf("OS image '%s' was not found in data centre '%s'", driver.ImageName, networkDomain.DatacenterID)
	}

	driver.ImageID = image.ID

	return nil
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

	log.Debugf("Deploying server '%s' ('%s')...", driver.ServerID, driver.MachineName)

	resource, err := client.WaitForDeploy(compute.ResourceTypeServer, driver.ServerID, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	server := resource.(*compute.Server)

	log.Debugf("Server '%s' ('%s') has been successfully deployed...", driver.ServerID, server.Name)

	driver.PrivateIPAddress = *server.Network.PrimaryAdapter.PrivateIPv4Address
	driver.IPAddress = driver.PrivateIPAddress // NAT rule not created yet.

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

// Has a NAT rule been created for the server?
func (driver *Driver) isNATRuleCreated() bool {
	return driver.NATRuleID != ""
}

// Create a NAT rule to expose the server.
func (driver *Driver) createNATRuleForServer() error {
	if !driver.isServerCreated() {
		return errors.New("Server has not been created")
	}

	if driver.isNATRuleCreated() {
		return fmt.Errorf("NAT rule '%s' has already been created for server '%s'", driver.NATRuleID, driver.MachineName)
	}

	log.Debugf("Creating NAT rule for server '%s' ('%s')...", driver.MachineName, driver.PrivateIPAddress)

	natRule, err := driver.getExistingNATRuleByInternalIP(driver.PrivateIPAddress)
	if natRule == nil {
		err = driver.ensurePublicIPAvailable()
		if err != nil {
			return err
		}

		client, err := driver.getCloudControlClient()
		if err != nil {
			return err
		}

		driver.NATRuleID, err = client.AddNATRule(driver.NetworkDomainID, driver.PrivateIPAddress, nil)
		if err != nil {
			return err
		}
		natRule, err = client.GetNATRule(driver.NATRuleID)
		if err != nil {
			return err
		}
		if natRule == nil {
			return fmt.Errorf("Failed to retrieve newly-created NAT rule '%s' for server '%s'", driver.NATRuleID, driver.MachineName)
		}

		log.Debugf("Created NAT rule '%s' for server '%s'", driver.NATRuleID)
	} else {
		driver.NATRuleID = natRule.ID

		log.Debugf("NAT rule already exists (Id = '%s').", driver.NATRuleID)
	}

	driver.IPAddress = natRule.ExternalIPAddress

	log.Debugf("Created NAT rule '%s' for server '%s' (Ext:'%s' -> Int:'%s').",
		driver.NATRuleID,
		driver.MachineName,
		driver.IPAddress,
		driver.PrivateIPAddress,
	)

	return nil
}

// Delete the the server's NAT rule (if any).
func (driver *Driver) deleteNATRuleForServer() error {
	if !driver.isServerCreated() {
		return errors.New("Server has not been created")
	}

	if !driver.isNATRuleCreated() {
		log.Debugf("Not deleting NAT rule for server '%s' (no NAT rule was created for it).")

		return nil
	}

	log.Debugf("Deleting NAT rule '%s' for server '%s' (Ext:'%s' -> Int:'%s')...", driver.NATRuleID, driver.MachineName, driver.IPAddress, driver.PrivateIPAddress)

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	natRule, err := client.GetNATRule(driver.NATRuleID)
	if err != nil {
		return err
	}
	if natRule == nil {
		log.Debugf("NAT rule '%s' not found; will treat it as already deleted.")

		driver.NATRuleID = ""

		return nil
	}

	err = client.DeleteNATRule(driver.NATRuleID)
	if err != nil {
		return err
	}

	log.Debugf("Deleted NAT rule '%s'.", driver.NATRuleID)

	driver.NATRuleID = ""
	driver.IPAddress = driver.PrivateIPAddress

	return nil
}

// Find the existing NAT rule (if any) that forwards IPv4 traffic to specified internal address.
func (driver *Driver) getExistingNATRuleByInternalIP(internalIPAddress string) (*compute.NATRule, error) {
	if driver.NetworkDomainID == "" {
		return nil, errors.New("Network domain has not been resolved.")
	}

	client, err := driver.getCloudControlClient()
	if err != nil {
		return nil, err
	}

	page := compute.DefaultPaging()
	for {
		var rules *compute.NATRules
		rules, err = client.ListNATRules(driver.NetworkDomainID, page)
		if err != nil {
			return nil, err
		}
		if rules.IsEmpty() {
			break // We're done
		}

		for _, rule := range rules.Rules {
			if rule.InternalIPAddress == internalIPAddress {
				return &rule, nil
			}
		}

		page.Next()
	}

	return nil, nil
}

// Ensure that at least one public IP address is available in the target network domain.
func (driver *Driver) ensurePublicIPAvailable() error {
	if driver.NetworkDomainID == "" {
		return errors.New("Network domain has not been resolved.")
	}

	log.Debugf("Verifying that network domain '%s' has a public IP available for server '%s'...", driver.NetworkDomainName, driver.MachineName)

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	availableIPs, err := client.GetAvailablePublicIPAddresses(driver.NetworkDomainID)
	if err != nil {
		return err
	}

	if len(availableIPs) == 0 {
		log.Debugf("There are no available public IPs in network domain '%s'; a new block of public IPs will be allocated.", driver.NetworkDomainID)

		blockID, err := client.AddPublicIPBlock(driver.NetworkDomainID)
		if err != nil {
			return err
		}

		log.Debugf("Allocated new public IP block '%s'.", blockID)
	}

	return nil
}

// Has a firewall rule been created to allow inbound SSH for the server?
func (driver *Driver) isSSHFirewallRuleCreated() bool {
	return driver.SSHFirewallRuleID != ""
}
