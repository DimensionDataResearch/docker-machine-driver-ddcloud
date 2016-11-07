# Contributing

Contributions are welcome, and they are greatly appreciated! Every little bit helps, and credit will always be given.

You can contribute in many ways:

## Types of Contributions

### Report Bugs

Report bugs at https://github.com/DimensionDataResearch/docker-machine-driver-ddcloud/issues.

If you are reporting a bug, please include:

* Your operating system name and version.
* Any details about your local setup that might be helpful in troubleshooting.
* Detailed steps to reproduce the bug.

### Fix Bugs

Look through the GitHub issues for bugs. Anything tagged with "bug"
and "help wanted" is open to whoever wants to implement it.

### Implement Features

Look through the GitHub issues for features. Anything tagged with "enhancement"
and "help wanted" is open to whoever wants to implement it.

### Write Documentation

The Docker Machine driver for CloudControl could always use more documentation, whether as part of the
official documentation, in GoDoc, or even on the web in blog posts, articles, and such.

### Submit Feedback

The best way to send feedback is to file an issue at https://github.com/DimensionDataResearch/docker-machine-driver-ddcloud/issues.

If you are proposing a feature:

* Explain in detail how it would work.
* Keep the scope as narrow as possible, to make it easier to implement.
* Remember that this is a volunteer-driven project, and that contributions are welcome :)

## Get Started!

Ready to contribute? Here's how to set up `docker-machine-driver-ddcloud` for local development.

1. Set up your build environment:
  * If you have Vagrant and VMWare / VirtualBox, you can run `vagrant up` to create a VM to work in.
    * The resulting VM will have your repository folder mapped into the VM as a shared folder (so you can use your regular editor on your machine, but run build commands in the VM).
    * Run `vagrant ssh` to connect to the VM.
  * Otherwise:
    1. Install Go version 1.6.x
    2. Make sure your GOPATH environment variable has been set.
    3. Run `go get -u github.com/DimensionDataResearch/docker-machine-driver-ddcloud`.
    4. Go to $GOPATH/src/github.com/DimensionDataResearch/docker-machine-driver-ddcloud.
2. Run `make dev` to build the provider, then `make install` to make it available to execute.

## Pull Request Guidelines

Before you submit a pull request, check that it meets these guidelines:

1. Your pull request should target the [development/v1.0](https://github.com/DimensionDataResearch/docker-machine-driver-ddcloud/tree/development/v1.0) branch.
2. The pull request should include tests (either unit or acceptance, as appropriate).
3. If the pull request adds or changes functionality, the docs should be updated.

## Tips

To enable extended logging of CloudControl API request and responses, set the `MCP_EXTENDED_LOGGING` environment variable to "1".
