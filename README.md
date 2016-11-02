# Docker Machine driver for Dimension Data CloudControl

## Usage

You will need:

* A network domain
* A VLAN in that network domain (servers will be attached to this VLAN)
* A firewall rule that permits SSH traffic from your (local) public IPv4 address to the VLAN's IPv4 network

The driver will allocate a public IP address and NAT rule for each machine that it creates.

### Example

```bash
docker-machine create --driver ddcloud \
	--ddcloud-region AU \
	--ddcloud-datacenter AU9 \
	--ddcloud-networkdomain 'my-docker-domain' \
	--ddcloud-vlan 'my-docker-vlan' \
	--ddcloud-ssh-key ~/.ssh/id_rsa \
	--ddcloud-ssh-bootstrap-password 'throw-away-password' \
	mydockermachine
```

If you're running on Windows, just remove the backslashes so the whole command is on a single line.

### Options

The driver supports all Docker Machine commands, and can be configured using the following command-line arguments (or environment variables):

* `ddcloud-user` - The user name used to authenticate to the CloudControl API.  
Environment: `DD_COMPUTE_USER`
* `ddcloud-password` - The password used to authenticate to the CloudControl API.  
Environment: `DD_COMPUTE_PASSWORD`.
* `ddcloud-region` - The CloudControl region name (e.g. AU, NA, EU, etc).  
Environment: `DD_COMPUTE_REGION`.
* `ddcloud-networkdomain` - The name of the target CloudControl network domain.
* `ddcloud-datacenter` - The name of the CloudControl datacenter (e.g. NA1, AU9) in which the network domain is located.
* `ddcloud-vlan` - The name of the target CloudControl VLAN.
* `ddcloud-ssh-user` - The SSH username to use.  
Default: "root".  
Environment: `DD_COMPUTE_SSH_USER`
* `ddcloud-ssh-key` - The SSH key file to use.  
Environment: `DD_COMPUTE_SSH_KEY`
* `ddcloud-ssh-port` - The SSH port to use.  
Default: 22.  
Environment: `DD_COMPUTE_SSH_PORT`
* `ddcloud-ssh-bootstrap-password` - The initial SSH password used to bootstrap SSH key authentication.  
This password is removed once the SSH key has been installed  
Environment: `DD_COMPUTE_SSH_BOOTSTRAP_PASSWORD`

## Installing the provider

Download the [latest release](https://github.com/DimensionDataResearch/docker-machine-driver-ddcloud/releases) and place the provider executable in the same directory as `docker-machine` executable (or somewhere on your `PATH`).

## Building the provider

If you'd rather run from source, simply run `make install` and you're good to go.
