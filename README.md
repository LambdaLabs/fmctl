# fmctl

A CLI tool to control NVIDIA fabricmanager

## Build

The `nvidia-fabricmanager-dev-<version>` package must be installed. We use cgo to compile, so the header files must be discoverable by `cc`. 

Run `make build` to build.