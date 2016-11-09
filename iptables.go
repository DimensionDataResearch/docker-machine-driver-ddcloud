package main

import (
	"fmt"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/ssh"
)

func (driver *Driver) clearIPTablesConfiguration() error {
	if !driver.isServerCreated() {
		return fmt.Errorf("Server '%s' has not been created", driver.MachineName)
	}

	log.Debugf("Flushing iptables configuration for server '%s'...",
		driver.MachineName,
	)

	client, err := ssh.NewNativeClient(driver.SSHUser, driver.IPAddress, driver.SSHPort, &ssh.Auth{
		Keys: []string{driver.SSHKeyPath},
	})
	if err != nil {
		return err
	}

	log.Debugf("Run 'iptables -F'...")
	output, err := client.Output(`iptables -F`)
	if err != nil {
		return fmt.Errorf("Failed to run 'iptables -F'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debugf("Successfully flushed iptables configuration for server '%s'.",
		driver.MachineName,
	)

	return nil
}
