# cf-ddns

A Dynamic DNS client for Cloudflare.

## Features

- Simple YAML configuration.
- Runs on defined interval (no need for cron scheduling).

## Installation

### go get
```bash
go get -u github.com/joshuar/cf-ddns
```

## Usage

Create a configuration file, see the example in this repo.  It should contain:

- Cloudflare account details:
  - Email username.
  - API Key.
  - Zone (i.e., DNS domain) in which records will be updated.
- Records to update, a list of hostnames for which the host running cf-ddns is called.
- Interval to run under, specified in a human way (i.e., 1h, 1d, 30m, etc.)
  - Interval is optional, default will be 1 hour.

Once you've got a configuration file, run the client:

```bash
./cf-ddns -c /path/to/config.yml
```

cf-ddns will continue to run on the interval defined, updating the records defined if your IP address changes.

## Contribution

I would welcome your contribution! If you find any improvement or issue you want to fix, feel free to send a pull request!

## Creator

[Joshua Rich](https://github.com/joshuar) (joshua.rich@gmail.com)

