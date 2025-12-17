# fmctl

A CLI tool to control NVIDIA fabricmanager

## Build

The `nvidia-fabricmanager-dev-<version>` package must be installed. We use cgo to compile, so the header files must be discoverable by `cc`. 

Run `make build` to build.

## Running

You can use fmctl to talk to the nv-fabricmanager service. Example:

```
./fmctl --address 127.0.0.1:6666 list
PARTITION ID  STATUS    GPUs  NVLINKS  GPU PHYSICAL IDs
------------  ------    ----  -------  ----------------
0             Inactive  8     144      1,2,3,4,5,6,7,8
1             Inactive  4     72       1,2,3,4
2             Inactive  4     72       5,6,7,8
3             Inactive  2     36       1,3
4             Inactive  2     36       2,4
5             Inactive  2     36       5,7
6             Inactive  2     36       6,8
7             Active    1     0        1
8             Active    1     0        2
9             Active    1     0        3
10            Active    1     0        4
11            Active    1     0        5
12            Active    1     0        6
13            Active    1     0        7
14            Active    1     0        8
```