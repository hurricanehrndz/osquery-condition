# osquery-condition

Runs queries through a running `osqueryd` and writes the results into Munki's
`/Library/Managed Installs/ConditionalItems.plist`, so manifests can use them as
[conditions](https://github.com/munki/munki/wiki/Conditional-Items). Run it as a
Munki preflight script. It assumes osquery is already installed and running.

## Usage

```
Usage of osquery-condition:
  -queries string
    	path to line delimited query file
  -socket string
    	path to osqueryd socket (default "/var/osquery/osquery.em")
  -version
    	print the version and exit
```

Example:

```sh
sudo osquery-condition -queries ./sample_queries
sudo cat '/Library/Managed Installs/ConditionalItems.plist'
```

## Writing queries

Put one query per line in a text file. Blank lines are skipped.

Each column in the results becomes a key named `osquery_<column>`. The value is
an array holding that column's value from every row of every query, so
`select * from system_info;` produces keys such as `osquery_hostname` and
`osquery_cpu_brand`. Keys written by other condition scripts are left alone.

If any query fails, the tool exits non-zero and leaves the plist unchanged.
Munki aborts its run when a preflight script exits non-zero.

## Install

Each [release](https://github.com/hurricanehrndz/osquery-condition/releases)
includes a signed and notarized universal package that installs
`/usr/local/bin/osquery-condition`.

## Development

Tools are pinned in `mise.toml` and `mise.lock`. Run `mise install`, then:

```sh
just test    # go vet + go test
just lint    # golangci-lint, formatting and go.mod checks
just fmt     # format Go sources
just hooks   # install the pre-commit hooks (prek)
just dist    # universal macOS binary in dist/
```

## Releasing

1. `just bump minor` (or `patch`/`major`) updates `main.go`, `embala.toml` and
   `versionbump.yaml`.
2. Merge the change to `master`, then push a `v<version>` tag.

The release workflow builds the universal binary, signs it, packages it with
[embala](https://github.com/hurricanehrndz/embala), signs and notarizes the
package, and publishes it to a GitHub release. It needs these repository
secrets:

- `MACOS_APPLICATION_CERTIFICATE_P12`, `MACOS_APPLICATION_CERTIFICATE_PASSWORD`
- `MACOS_INSTALLER_CERTIFICATE_P12`, `MACOS_INSTALLER_CERTIFICATE_PASSWORD`
- `MACOS_NOTARY_API_KEY_JSON` (from `rcodesign encode-app-store-connect-api-key`)
