# Network Policy Tester

`npt` is a command line tool to evaluate `CiliumNetworkPolicies` and `CiliumClusterwideNetworkPolicies` against different scenarios without the need to bootstrap a cluster with Cilium CRDs.

## Installation

Download pre-compiled binaries in [Releases](https://github.com/Hsn723/npt/releases) or install manually:

```sh
go install github.com/hsn723/npt@latest
```

## Usage

```sh
Usage:
  npt [flags]
  npt [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print the version information

Flags:
  -h, --help                    help for npt
      --json                    Output results in JSON format
  -o, --output string           File to write output results to
      --policy-dir strings      Directories containing policy files
      --policy-file strings     Policy files to load
      --scenario-dir strings    Directories containing scenario files
      --scenario-file strings   Scenario files to load
  -v, --verbose                 Enable verbose output
```

`CiliumNetworkPolicies` and `CiliumClusterwideNetworkPolicies` locations are specified via the `--policy-dir` and/or `--policy-file` command-line flags, and all policies found will be loaded. Test scenario locations are specified via the `--scenario-dir` and/or `scenario-file` command-line flags. Each scenario will be run against the loaded policies, and results are displayed at the end.

Test scenarios are specified via a JSON file. See the examples directory for an example scenario file.
