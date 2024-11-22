# go-mpd-discplayer

[![Go Reference](https://pkg.go.dev/badge/github.com/b0bbywan/go-mpd-discplayer.svg)](https://pkg.go.dev/github.com/b0bbywan/go-mpd-discplayer)

`mpd-discplayer` is a Go-based client designed for seamless disc playback with an MPD (Music Player Daemon) server. It automatically handles disconnections and uses [go-disc-cuer](https://github.com/b0bbywan/go-disc-cuer/) to generate CUE files for inserted audio discs.


## Features

- **Disc Playback Automation**: Monitors for inserted audio discs and generates CUE files using go-disc-cuer.
- **Reconnection Logic**: Automatically reconnects to the MPD server if the connection is lost.
- **Configurable Settings**: Supports configuration via a unified config file or environment variables.
- **Reliable Connection Management**: Thread-safe operations to manage MPD client connections and operations.


## Installation

To use `mpd-discplayer`, you'll need to have Go installed on your system. If you don’t have Go installed, follow the instructions at [https://golang.org/doc/install](https://golang.org/doc/install).

### Clone the repository

```bash
git clone https://github.com/b0bbywan/go-mpd-discplayer.git
cd go-mpd-discplayer
```

### Install dependencies
Ensure libgudev and libdiscid are installed. Run the following commands based on your OS:

```bash
# Debian
sudo apt install libdiscid0 libdiscid-dev libgudev-1.0-0 libgudev-1.0-dev
# Fedora
sudo dnf install libdiscid libdiscid-devel libgudev libgudev-devel
```

### Build the Project
```bash
go build -o mpd-discplayer
```


## Usage

Simply run the program. mpd-discplayer will:

- Monitor the system for an audio disc insertion.
- Automatically use go-disc-cuer to generate a CUE file.
- Play the disc on the MPD server.

```bash
./mpd-discplayer
```


## Configuration

`mpd-discplayer` can be configured using a YAML configuration file or environment variables. This allows flexibility in managing settings for both the MPD server connection and the `disc-cuer` tool. Below is a detailed explanation of the configuration options and how to use them.
Visit [go-disc-cuer](https://github.com/b0bbywan/go-disc-cuer/) for more informations

### Configuration File

The configuration file is expected in one of the following locations:
1. `/etc/disc-cuer/config.yml` (system-wide configuration)
2. `~/.config/disc-cuer/config.yml` (user-specific configuration)

The file should be written in YAML format, and a sample structure is shown below:

```yaml
gnuHelloEmail: "your-email@example.com"
gnuDbUrl: "https://gnudb.gnudb.org"
cacheLocation: "/var/cache/disc-cuer"

MPDConnection:
  Type: "tcp"
  Address: "127.0.0.1:6600"
  ReconnectWait: 30

TargetDevice: "sr0"
```

#### MPD Connection Options
Under the MPDConnection key, you configure how mpd-discplayer connects to the MPD server:

- **Type**:
The type of connection to use. Supported values:
	- `"unix"`: For a Unix socket connection.
	- `"tcp"`: For a TCP connection over the network. (default)
- **Address**:
	- For Type: `"unix"`, this is the path to the MPD socket file (e.g., `/var/run/mpd/socket`).
	- For Type: `"tcp"`, this is the <hostname>:<port> of the MPD server (e.g., `127.0.0.1:6600`) *(default)*.

### Environment Variables

If a configuration file is not provided, you can use environment variables to set the same options. Below is the list of supported variables and their defaults (if applicable):

| Environment Variable           | YAML equivalent                          | Default Value                  |
|--------------------------------|--------------------------------------|--------------------------------|
| `DISC_CUER_GNUHELLOEMAIL`     | `gnuHelloEmail`.      | *(no default,  empty value disable the integration)*      |
| `DISC_CUER_GNUDBURL`          | `gnuDbUrl`.           | `http://gnudb.gnudb.org`    |
| `DISC_CUER_CACHELOCATION`     | `cacheLocation`.      | `/var/cache/disc-cuer` *for root* / `~/.cache/disc-cuer` *for standard users*      |
| `MPD_DISCPLAYER_TYPE`         | `MPDConnection.Type`. | `tcp`                        |
| `MPD_DISCPLAYER_ADDRESS`      | `MPDConnection.Address`. | `127.0.0.1:6600`   |
| `MPD_DISCPLAYER_RECONNECT_WAIT`      | `MPDConnection.ReconnectWait`. | `30` (in seconds)          |
| `MPD_DISCPLAYER_TARGET_DEVICE` | `TargetDevice` | `sr0`


#### Priority of Configuration
The configuration is loaded in the following order of priority:
- Environment variables (highest priority)
- User-specific configuration file (~/.config/disc-cuer/config.yml)
- System-wide configuration file (/etc/disc-cuer/config.yml)

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing
Contributions are welcome! Feel free to fork this repository, make changes, and create a pull request.

Acknowledgments
Thanks to [gompd](https://github.com/fhs/gompd) for the underlying MPD client implementation.
