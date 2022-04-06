<p align="center">
    <img width=500 src="logo_full.png?raw=true">
</p>

Emulate `ssh -D` behavior even if `AllowTcpForwarding` is disabled by administrator in `sshd_config`. This tool creates 
the tunnel sending serialized data through STDIN to a remote process (which is running a Socks5 Server) and receiving
the response trought STDOUT on a normal SSH session channel.

### Authors
* **Raúl Sampedro** - [@rsrdesarrollo](https://www.linkedin.com/in/rsrdesarrollo/) - Initial Work

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing 
purposes.

### Prerequisites

You only need [golang (>=1.10)](https://golang.org/dl/) to build this tool.

### Installing

Just install the command line tool

```
go install github.com/rsrdesarrollo/SaSSHimi@latest
```

### Usage

Just run it as a normal ssh client

```
SaSSHimi server user@localhost
```

You can fing more help using `--help`

```
$ SaSSHimi server --help

Run local server to create tunnels

Usage:
  SaSSHimi server <user@host:port|host_id> [flags]

Flags:
      --bind string                Help message for toggle (default "127.0.0.1:1080")
  -h, --help                       help for server
  -i, --identity_file string       Path to private key
      --remote_executable string   Path to SaSSHimi to run on remote machine

Global Flags:
      --config string   config file (default is $HOME/.SaSSHimi.yaml)
  -v, --verbose count   verbose level
```

### Configuration File

Like SSH, SaSSHimi has a configuration file where you can set some basic config for your most common hosts.
You can find a sample of the syntax of this file in [config_sample.yml](config_sample.yml).

By default SaSSHimi try to find this config file at `~/.SaSSHimi.yaml`. You can change this behaviour by using the 
`--config` flag.

**ONLY USE PASSWORDS IN THE CONFIG AT YOUR OWN RISK**

### TODO

- [x] Support Public key authentication.
- [ ] Support Enc Private Keys.
- [x] Improve configuration file.
- [x] Add more command options to control binding ports.
- [ ] Implement known_hosts support

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of 
conduct, and the process for submitting pull requests to this project.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the 
[tags on this repository](https://github.com/rsrdesarrollo/SaSSHimi/tags).

## License

This project is licensed under the Apache License Version 2.0- see the [LICENSE](LICENSE) file for details

## Acknowledgments

-  [@maramarillophotography](https://www.instagram.com/maramarillophotography/) for such an amazing logo ;)
