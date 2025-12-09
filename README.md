# Terraform Provider for MetaKube

- Documentation: https://registry.terraform.io/providers/syseleven/metakube/latest/docs
- Check the [examples](./examples) for quick start.

## Overview

- [Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
- [Go](https://golang.org/doc/install) 1.12 to build the provider plugin


## Troubleshooting

If you encounter issues [file an issue][1]

### Before you start

* Please familiarize yourself with the [Code of Conduct][4] before contributing.
* See [CONTRIBUTING.md][2] for instructions on the developer certificate of origin that we require.

### Pull requests

* We welcome pull requests. Feel free to dig through the [issues][1] and jump in.

### Acceptance tests

Acceptance tests are executed automatically in a mirroring repository on gitlab.

To run tests locally, copy `env-template.sh` to `.env` and set the values accordingly.

### Debugging

If you haven't yet, create a ~/.terraformrc that overrides the `syseleven/metakube` provider reference so it uses the binary we just built.
This needs to be done only once.

```
./provider-debug/setup.sh
```

Now you can invoke Teraform normally from any directory.

This will use the just built binary for the provider.

If you want to run the binary under dlv, set `DEBUG` to `true` in the shell.
This will stop whenever the provider binary is invoked, waiting for a debug client (e.g. dlv, or an IDE) to connect on localhost port 2345.

```
DEBUG=true terraform apply
```

To stop using the debug binary and use the normal upstream version of the provider again, remove or rename `~/.terraformrc`.

## Changelog

See [the list of releases][3] to find out about feature changes.

[1]: https://github.com/syseleven/terraform-provider-metakube/issues
[2]: https://github.com/syseleven/terraform-provider-metakube/blob/syseleven/master/CONTRIBUTING.md
[3]: https://github.com/syseleven/terraform-provider-metakube/releases
[4]: https://github.com/syseleven/terraform-provider-metakube/blob/syseleven/master/code-of-conduct.md
