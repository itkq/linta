# linta

GitHub Actions' permissions linter.

https://user-images.githubusercontent.com/8341422/221356688-69144e84-1e43-46bb-a642-59bc4d8feff3.mov

## Usage

```
$ linta help
NAME:
   linta - A linter for GitHub Actions' permissions

USAGE:
   linta [global options] command [command options] [arguments...]

COMMANDS:
   init     Initialize config file
   run      Run linter
   version  Version of a tool
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug     Enable debug mode (default: false)
   --help, -h  show help

$ linta init --help
NAME:
   linta init - Initialize config file

USAGE:
   linta init [command options] [arguments...]

OPTIONS:
   --output-path PATH, -o PATH  Write configuration to PATH
   --overwrite                  Overwrite existing configuration file (default: false)
   --help, -h                   show help

$ linta run --help
NAME:
   linta run - Run linter

USAGE:
   linta run [command options] [arguments...]

OPTIONS:
   --config-path PATH, -c PATH  Load configuration from PATH
   --format value, -f value     Output format. One of: [json, text] (default: "text")
   --help, -h
```

## How it works

Just compare the permissions by knowledge-based config like:

```yaml
repositories:
  "actions/checkout":
    contents: "read"
```

There's no magic!

## TODO
- Support workflow-level permissions
