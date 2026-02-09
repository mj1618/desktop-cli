# Desktop CLI

A command-line tool for desktop automation.

## Installation

### Download Binary (Recommended)

Download the latest binary for your platform from the [releases page](https://github.com/mj1618/desktop-cli/releases/latest).

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_darwin_arm64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_darwin_amd64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

**Linux (x64):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_linux_amd64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

**Linux (ARM64):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_linux_arm64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

### Install with Go

If you have Go installed:

```bash
go install github.com/mj1618/desktop-cli@latest
```

### Build from Source

```bash
git clone https://github.com/mj1618/desktop-cli.git
cd desktop-cli
go build -o desktop-cli .
sudo mv desktop-cli /usr/local/bin/
```

## Development

### Build

```bash
go build -v ./...
```

### Test

```bash
go test -v ./...
```

## License

MIT
