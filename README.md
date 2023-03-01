# cf-ddns

![Apache 2.0](https://img.shields.io/github/license/joshuar/cf-ddns) 
![GitHub last commit](https://img.shields.io/github/last-commit/joshuar/cf-ddns)
[![Go Report Card](https://goreportcard.com/badge/github.com/joshuar/cf-ddns?style=flat-square)](https://goreportcard.com/report/github.com/joshuar/cf-ddns) 
[![Go Reference](https://pkg.go.dev/badge/github.com/joshuar/cf-ddns.svg)](https://pkg.go.dev/github.com/joshuar/cf-ddns)
[![Release](https://img.shields.io/github/release/joshuar/cf-ddns.svg?style=flat-square)](https://github.com/joshuar/cf-ddns/releases/latest)

## What is it?

A [Dynamic DNS](https://en.wikipedia.org/wiki/Dynamic_DNS) client for
[Cloudflare](https://www.cloudflare.com/dns/). cf-ddns will update any A/AAAA
(IPv4 and IPv6) records for the listed hostnames defined in the configuration
file.  

## Features

- Simple (YAML) configuration. Just specify your account details, domain and a
  list of records to update.  
- Runs on a defined interval (no need for cron scheduling) or as a one-shot execution.
- Fail-over external IP service checks.  

## Installation

### Linux Packages

RPM/DEB packages are available, see the [releases](https://github.com/joshuar/cf-ddns/releases) page.

### go get
```shell
go get -u github.com/joshuar/cf-ddns
```

## Usage

Create a configuration file; see the example in this repo.  It should contain:

- Cloudflare account details:
  - Email username.
  - API Key.
  - Zone (i.e., DNS domain) in which record updates are made.
- Records to update, a list of hostnames for which the host running cf-ddns is
  called.
- Interval to run under, specified in a human way (i.e., `1h`, `1d`, `30m`, etc.)
  - Interval is optional; the default will be 1 hour.

cf-ddns looks for a configuration file at `/etc/cf-ddns/cf-ddns.yaml` by default, but you
can specify a path with the `--config` command-line option.

Once you've got a configuration file, you have two options to run:

### Daemon Mode

In daemon mode, cf-ddns will run constantly and check on the interval specific in the configuration
file whether an update is needed:

```bash
cf-ddns daemon # --config /path/to/config.yml (optional)
```

A systemd service file has been provided that will run cf-ddns in daemon mode using a configuration
file at the default location (`/etc/cf-ddns/cf-ddns.yaml`).


### One-shot Mode

In one-shot mode, cf-ddns will run once, will perform updates as needed, then exit:

```bash
cf-ddns # --config /path/to/config.yml (optional)
```

One-shot mode can be run on a systemd timer, see the provided `cf-ddns-oneshot.{service,timer}`
files. 

## Contributions

I would welcome your contribution! If you find any improvement or issue you want
to fix, feel free to send a pull request!

## Creator

[Joshua Rich](https://github.com/joshuar) (joshua.rich@gmail.com)
