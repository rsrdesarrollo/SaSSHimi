<p align="center">
    <img width=500 src="logo_full.png?raw=true">
</p>

Emulate `ssh -D` behavior even if `AllowTcpForwarding` is disabled by administrator in `sshd_config`. This tool creates 
the tunnel sending serialized data through STDIN to a remote process (which is running a Socks5 Server) and receiving
the response trought STDOUT on a normal SSH session channel.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing 
purposes.

### Prerequisites

You only need [golang](https://golang.org/dl/) to use this tool.

### Installing

Just install the command line tool

```
go install  github.com/rsrdesarrollo/ssh-tunnel
```

### Usage

*TBD*

### TODO

* Support Public key authentication.
* Improve configuration file.
* Add more command options to control binding ports.

## Contributing

Please read [CONTRIBUTING.md](https://github.com/rsrdesarrollo/ssh-tunnel/contributors) for details on our code of 
conduct, and the process for submitting pull requests to this project.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the 
[tags on this repository](https://github.com/your/project/tags).

## Authors

* **Raúl Sampedro** - *Initial work* - [@rsrdesarrollo](https://github.com/rsrdesarrollo)

## License

This project is licensed under the Apache License Version 2.0- see the [LICENSE](LICENSE) file for details
