package main

/*
 * SSH key bootstrapper
 * --------------------
 *
 * Installs an SSH key onto the target machine so the rest of docker-machine can do its stuff (then disables password authentication).
 *
 * This is required because CloudControl only supports specifying passwords during server deployment (not SSH keys).
 */

import (
	"errors"
	"fmt"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/docker/machine/libmachine/ssh"
	"io/ioutil"
	"os"
	"path"
)

// Bootstrap key-based SSH authentication by installing an SSH public key on the target machine.
func (driver *Driver) installSSHKey() error {
	if driver.ServerID == "" {
		return errors.New("Server has not been deployed")
	}

	log.Debugf("Starting SSH bootstrap process (as user '%s') for target host '%s:%d'...",
		driver.SSHUser,
		driver.IPAddress,
		driver.SSHPort,
	)

	// We explicitly need the native client because we're using password authentication.
	client, err := ssh.NewNativeClient(driver.SSHUser, driver.IPAddress, driver.SSHPort, &ssh.Auth{
		Passwords: []string{driver.SSHBootstrapPassword},
	})
	if err != nil {
		return err
	}

	log.Debugf("Create '$HOME/.ssh'...")
	output, err := client.Output(`mkdir -p "$HOME/.ssh"`)
	if err != nil {
		return fmt.Errorf("Failed to create '$HOME/.ssh'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debugf("Secure '$HOME/.ssh'...")
	output, err = client.Output(`chmod 700 "$HOME/.ssh"`)
	if err != nil {
		return fmt.Errorf("Failed to secure '$HOME/.ssh'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debugf("Add SSH key to '$HOME/.ssh/authorized_keys'...")
	publicKey, err := driver.getSSHPublicKey()
	if err != nil {
		return err
	}
	output, err = client.Output(fmt.Sprintf(
		`echo '%s' >> "$HOME/.ssh/authorized_keys"`, publicKey,
	))
	if err != nil {
		return fmt.Errorf("Failed to add SSH key to '$HOME/.ssh/authorized_keys'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debugf("Secure '$HOME/.ssh/authorized_keys'...")
	output, err = client.Output(`chmod 600 "$HOME/.ssh/authorized_keys"`)
	if err != nil {
		return fmt.Errorf("Failed to secure '$HOME/.ssh/authorized_keys'\n%s\nOutput:\n%s",
			err.Error(),
			output,
		)
	}

	log.Debugf("Remove password for '%s'...", driver.SSHUser)
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

	driver.SSHBootstrapPassword = ""

	log.Debugf("SSH bootstrap process complete; the public key from '%s' is now installed on host '%s:%d' for user '%s'.",
		driver.SSHKeyPath+".pub",
		driver.IPAddress,
		driver.SSHPort,
		driver.SSHUser,
	)

	return nil
}

// Import the configured SSH key files into the machine store folder.
func (driver *Driver) importSSHKey() error {
	if driver.SSHKey == "" {
		return errors.New("SSH key path not configured")
	}

	driver.SSHKeyPath = driver.ResolveStorePath(
		path.Base(driver.SSHKey),
	)
	err := copySSHKey(driver.SSHKey, driver.SSHKeyPath)
	if err != nil {
		log.Infof("Couldn't copy SSH private key : %s", err.Error())

		return err
	}

	err = copySSHKey(driver.SSHKey+".pub", driver.SSHKeyPath+".pub")
	if err != nil {
		log.Infof("Couldn't copy SSH public key : %s", err.Error())

		return err
	}

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

// Copy an SSH key file.
func copySSHKey(sourceFile string, destinationFile string) error {
	err := mcnutils.CopyFile(sourceFile, destinationFile)
	if err != nil {
		return fmt.Errorf("unable to copy ssh key: %s", err.Error())
	}

	err = os.Chmod(destinationFile, 0600)
	if err != nil {
		return fmt.Errorf("unable to set permissions on the ssh key: %s", err.Error())
	}

	return nil
}
