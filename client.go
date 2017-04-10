package main

/*
 * Driver support for the CloudControl API client
 * ----------------------------------------------
 */

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/docker/machine/libmachine/log"
)

// CloudControl client retry
const (
	// The maximum number of times the client will retry in the case of a network error connecting to the CloudContron API.
	clientMaxRetry = 5

	// The period of time between retries against the CloudControl API.
	clientRetryPeriod = 5 * time.Second
)

// Timeouts
const (
	// CloudControl server deployment timeout.
	serverCreateTimeout = 15 * time.Minute

	// CloudControl resource deletion timeout.
	serverDeleteTimeout = 10 * time.Minute

	// CloudControl server startup timeout.
	serverStartTimeout = 3 * time.Minute

	// CloudControl server shutdown timeout.
	serverStopTimeout = 3 * time.Minute

	// CloudControl server power-off timeout.
	serverPowerOffTimeout = 2 * time.Minute
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

	if driver.CloudControlRegion != "" {
		client = compute.NewClient(driver.CloudControlRegion, driver.CloudControlUser, driver.CloudControlPassword)
	} else if driver.CloudControlEndPointURI != "" {
		client = compute.NewClientWithBaseAddress(driver.CloudControlEndPointURI, driver.CloudControlUser, driver.CloudControlPassword)
	} else {
		err = errors.New("Cannot connect to CloudControl API (neither region nor custom end-point URI have been configured)")

		return
	}
	client.ConfigureRetry(clientMaxRetry, clientRetryPeriod)

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
		return nil, fmt.Errorf("Server '%s' has not been created", driver.MachineName)
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

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	image, err := client.FindOSImage(driver.ImageName, driver.DataCenterID)
	if err != nil {
		return err
	}
	if image == nil {
		log.Errorf("OS image '%s' was not found in data centre '%s'.", driver.ImageName, driver.DataCenterID)

		return fmt.Errorf("OS image '%s' was not found in data centre '%s'", driver.ImageName, driver.DataCenterID)
	}

	if image.OperatingSystem.Family != "UNIX" {
		return fmt.Errorf("OS image '%s' in data centre '%s' is not from a supported OS family (expected 'UNIX', but found '%s')",
			driver.ImageName,
			driver.DataCenterID,
			image.OperatingSystem.Family,
		)
	}

	driver.ImageID = image.ID
	driver.ImageOSType = image.OperatingSystem.ID

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

	resource, err := client.WaitForDeploy(compute.ResourceTypeServer, driver.ServerID, serverCreateTimeout)
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

// Start the target server.
func (driver *Driver) startServer() error {
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

	_, err = client.WaitForChange(compute.ResourceTypeServer, driver.ServerID, "Start server", serverStartTimeout)

	return err
}

// Stop the target server.
func (driver *Driver) stopServer() error {
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

	_, err = client.WaitForChange(compute.ResourceTypeServer, driver.ServerID, "Shut down server", serverStopTimeout)

	return err
}

// Stop the target server.
func (driver *Driver) powerOffServer() error {
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

	_, err = client.WaitForChange(compute.ResourceTypeServer, driver.ServerID, "Power off server", serverPowerOffTimeout)

	return err
}

// Has a NAT rule been created for the server?
func (driver *Driver) isNATRuleCreated() bool {
	return driver.NATRuleID != ""
}

// Create a NAT rule to expose the server.
func (driver *Driver) createNATRuleForServer() error {
	if !driver.isServerCreated() {
		return fmt.Errorf("Server '%s' has not been created", driver.MachineName)
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
		return fmt.Errorf("Server '%s' has not been created", driver.MachineName)
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

// Create a firewall rule to enable inbound SSH connections to the target server from the client machine's (external) IP address.
func (driver *Driver) createSSHFirewallRule() error {
	if !driver.isServerCreated() {
		return fmt.Errorf("Server '%s' has not been created", driver.MachineName)
	}

	if driver.isSSHFirewallRuleCreated() {
		return fmt.Errorf("Firewall rule '%s' has already been created for server '%s'", driver.SSHFirewallRuleID, driver.MachineName)
	}

	log.Debugf("Creating SSH firewall rule for server '%s' (allow inbound traffic on port %d from '%s' to '%s')...",
		driver.MachineName,
		driver.SSHPort,
		driver.ClientPublicIPAddress,
		driver.IPAddress,
	)

	ruleConfiguration := compute.FirewallRuleConfiguration{
		Name:            driver.buildFirewallRuleName("SSH"),
		NetworkDomainID: driver.NetworkDomainID,
	}
	ruleConfiguration.Accept()
	ruleConfiguration.Enable()
	ruleConfiguration.IPv4()
	ruleConfiguration.TCP()
	ruleConfiguration.MatchSourceAddress(driver.ClientPublicIPAddress)
	ruleConfiguration.MatchDestinationAddress(driver.IPAddress)
	ruleConfiguration.MatchDestinationPort(driver.SSHPort)
	ruleConfiguration.PlaceFirst()

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	firewallRuleID, err := client.CreateFirewallRule(ruleConfiguration)
	if err != nil {
		return err
	}

	driver.SSHFirewallRuleID = firewallRuleID

	log.Debugf("Created SSH firewall rule '%s' for server '%s'.", driver.SSHFirewallRuleID, driver.ServerID)

	return nil
}

// Delete the firewall rule that enables inbound SSH connections to the target server from the client machine's (external) IP address.
func (driver *Driver) deleteSSHFirewallRule() error {
	if !driver.isServerCreated() {
		return fmt.Errorf("Server '%s' has not been created", driver.MachineName)
	}

	if !driver.isSSHFirewallRuleCreated() {
		return fmt.Errorf("Firewall rule has not been created for server '%s'", driver.MachineName)
	}

	log.Debugf("Deleting SSH firewall rule '%s' for server '%s'...",
		driver.MachineName,
		driver.SSHFirewallRuleID,
	)

	client, err := driver.getCloudControlClient()
	if err != nil {
		return err
	}

	err = client.DeleteFirewallRule(driver.SSHFirewallRuleID)
	if err != nil {
		return err
	}

	log.Debugf("Deleted firewall rule '%s'.", driver.SSHFirewallRuleID)

	driver.SSHFirewallRuleID = ""

	return nil
}

// Name sanitiser for firewall rules.
var firewallRuleNameSanitizer = strings.NewReplacer("-", ".", "_", ".")

// Build an acceptable name for a firewall rule.
func (driver *Driver) buildFirewallRuleName(suffix string) string {
	return strings.ToLower(
		firewallRuleNameSanitizer.Replace(driver.MachineName) + "." + suffix,
	)
}
