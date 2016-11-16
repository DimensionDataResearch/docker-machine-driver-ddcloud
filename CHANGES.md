# Changes

## v0.8

Breaking changes:

* New features:

* The driver can now use the new `--ddcloud-mcp-endpoint` command-line argument (environment: `MCP_ENDPOINT`) to designate a custom end-point URI for the CloudControl API.

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
