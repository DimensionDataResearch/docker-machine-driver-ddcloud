# Changes

## v0.9.3

Enhancements:

* The driver now creates a firewall rule to permit Docker API access if `--ddcloud-create-docker-firewall-rule` is specified.

## v0.9.2

Enhancements:

* Can now specify a custom private IPv4 address for the server.

## v0.9.1

Bug fixes:

* Specify firewall rule placement when exposing deployed server.

## v0.9

New features:

* More useful help / documentation for auto-generated SSH keys.

## v0.8

New features:

* The driver can now use the new `--ddcloud-mcp-endpoint` command-line argument (environment: `MCP_ENDPOINT`) to designate a custom end-point URI for the CloudControl API.
* The driver will now generate a new SSH keypair if one was not already configured via command-line arguments.

Breaking changes:

* The following command-line arguments have changed to be consistent with their corresponding environment variables:
  * `--ddcloud-user` is now `--ddcloud-mcp-user`
  * `--ddcloud-password` is now `--ddcloud-mcp-password`
  * `--ddcloud-region` is now `--ddcloud-mcp-region`

## v0.7

New features:

* Enable explicitly specifying the client's public IP address via `--ddcloud-client-public-ip` (#7).

## v0.6

New features:

* Add support for using private IP addresses instead of public ones (#6)

## v0.5

Bug fixes:

* Fixed error about missing network domain Id when creating SSH firewall rules (#7).

## v0.4

New features:

* Add support for using private IP addresses instead of public ones (#6)

## v0.4

Breaking changes:

* All environment variables that previously used the prefix `DD_COMPUTE_` have been changed to use the prefix `MCP_`.
