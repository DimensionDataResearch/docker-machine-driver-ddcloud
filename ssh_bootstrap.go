package main

import (
	"errors"
	"fmt"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/ssh"
	"io/ioutil"
	"os"
)

// Bootstrap SSH by installing an SSH public key on the target machine.
func (driver *Driver) installSSHKey() error {
	if driver.ServerID == "" {
		return errors.New("Server has not been deployed")
	}

	log.Debug("Starting SSH bootstrap process (as user '%s') for target host '%s:%d'...",
		driver.SSHUser,
		driver.IPAddress,
		driver.SSHPort,
	)

	client, err := ssh.NewClient(driver.SSHUser, driver.IPAddress, driver.SSHPort, &ssh.Auth{
		Passwords: []string{driver.SSHBootstrapPassword},
	})
	if err != nil {
		return err
	}

	log.Debug("Create '~/.ssh'...")
	output, err := client.Output(`mkdir -p "~/.ssh"`)
	if err != nil {
		return fmt.Errorf("Failed to create '~/.ssh'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debug("Ensure '~/.ssh/authorized_keys' exists...")
	output, err = client.Output(`touch "~/.ssh/authorized_keys"`)
	if err != nil {
		return fmt.Errorf("Failed to create '~/.ssh/authorized_keys'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debug("Ensure '~/.ssh/authorized_keys' exists...")
	output, err = client.Output(`touch "~/.ssh/authorized_keys"`)
	if err != nil {
		return fmt.Errorf("Failed to create '~/.ssh/authorized_keys'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	publicKey, err := driver.getSSHPublicKey()
	if err != nil {
		return err
	}

	log.Debug("Add SSH key to '~/.ssh/authorized_keys'...")
	output, err = client.Output(fmt.Sprintf(
		`echo "%s" >> "~/.ssh/authorized_keys"`, publicKey,
	))
	if err != nil {
		return fmt.Errorf("Failed to create '~/.ssh/authorized_keys'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debug("Secure '~/.ssh'...")
	output, err = client.Output(`chmod -R 700 "~/.ssh"`)
	if err != nil {
		return fmt.Errorf("Failed to secure '~/.ssh'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debug("Remove password for '%s'...", driver.SSHUser)
	output, err = client.Output(fmt.Sprintf(
		"passwd -d %s", driver.SSHUser,
	))
	if err != nil {
		return fmt.Errorf("Failed to remove initial password for '%s'\n%s\nOutput:\n%s",
			driver.SSHUser,
			err.Error(),
			output,
		)
	}

	log.Debug("SSH bootstrap process complete; the public key from '%s' is now installed on host '%s:%d' for user '%s'.",
		driver.SSHKeyPath+".pub",
		driver.IPAddress,
		driver.SSHPort,
		driver.SSHUser,
	)

	return nil
}

// Get the public portion of the configured SSH key.
func (driver *Driver) getSSHPublicKey() (string, error) {
	publicKeyFile, err := os.Open(driver.SSHKeyPath + ".pub")
	if err != nil {
		return "", err
	}
	defer publicKeyFile.Close()

	publicKeyData, err := ioutil.ReadAll(publicKeyFile)
	if err != nil {
		return "", err
	}

	return string(publicKeyData), nil
}
