# Unified Clustering Tools (UCL)

> Unified Clustering Tools (UCL) is a framework for ensuring the consistency of the cluster application.
> This framework launch applications on multiple Socs/VMs/PCs. It ensures the startup order of related processes, monitors their operational status, and reliably reclaims resources when they are no longer needed.
> UCL is divided into Master and Worker. 
> Worker runs constantly on each host and waiting for the connection from Master and Master Communicates application execution information to Worker.

## Contents

- [Unified Clustering Tools (UCL)](#architecture)
  - [Contents](#contents)
  - [Repository structure](#repository-structure)
  - [How to install](#how-to-install)
    - [Golang setup](#golang-setup)
    - [Build UCL framework](#build-ucl-framework)
  - [How to Use](#how-to-use)
    - [Json settings](#json-settings)
    - [Workers side](#workers-side)
    - [Master side](#master-side)
  - [Colaborate with RVGPU](#colaborate-with-rvgpu)
    - [How to install RVGPU](#how-to-install-rvgpu)
    - [How to install rvgpu-wlproxy](#how-to-install-rvgpu-wlproxy)
    - [How to run UCL](#how-to-run-ucl)
      - [Run ucl-virtio-gpu-wl-recv](#run-ucl-virtio-gpu-wl-recv)
      - [Run ucl-virtio-gpu-wl-send](#run-ucl-virtio-gpu-wl-send)
      - [Using Json configuration](#using-json-configuration)
        - [Json settings](#json-settings)
        - [Workers side](#workers-side)
        - [Master side](#master-side)

## Repository structure

```
.
├── cmd
│   ├── Makefile
│   ├── ucl-consistency-keeper
│   │   ├── Makefile
│   │   └── ucl_consistency_keeper.go
│   ├── ucl-distrib-com
│   │   ├── Makefile
│   │   └── ucl_distrib_com.go
│   ├── ucl-launcher
│   │   ├── Makefile
│   │   └── ucl_launcher.go
│   ├── ucl-ncount-master
│   │   ├── Makefile
│   │   └── ucl_ncount_master.go
│   ├── ucl-ncount-worker
│   │   ├── Makefile
│   │   └── ucl_ncount_worker.go
│   ├── ucl-nkeep-master
│   │   ├── Makefile
│   │   └── ucl_nkeep_master.go
│   ├── ucl-nkeep-worker
│   │   ├── Makefile
│   │   └── ucl_nkeep_worker.go
│   ├── ucl-timing-wrapper
│   │   ├── Makefile
│   │   └── ucl_timing_wrapper.go
│   ├── ucl-virtio-gpu-wl-recv
│   │   ├── Makefile
│   │   └── ucl_virtio_gpu_wl_recv.go
│   └── ucl-virtio-gpu-wl-send
│       ├── Makefile
│       └── ucl_virtio_gpu_wl_send.go
├── example
│   ├── multi-node
│   │   ├── app.json
│   │   └── virtual-screen-def.json
│   └── single-node
│       ├── app.json
│       └── virtual-screen-def.json
├── internal
│   ├── Makefile
│   ├── ucl
│   │   ├── common_settings.go
│   │   ├── ucl_node_conn.go
│   │   ├── ucl_proc_sighandler.go
│   │   └── virtual_screen_def.go
│   └── ulog
│       ├── Makefile
│       └── ulog.go
├── LICENSE.md
├── Makefile
└── README.md
```


# How to install
## Golang setup
Before building UCL API, you need to install and configure Golang.
```
sudo apt-get install golang
export GOPATH=<your go work directory>
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
```

## Build UCL framework
You can easily build UCL framweork using make.
```
mkdir -p $GOPATH/src
cd $GOPATH/src
git clone https://github.com/unified-hmi/ucl-tools.git
cd ucl-tools
make
```

# How to use
UCL is possible to launch applications on multiple SoCs/VMs/PCs, control the launch timing and check for survival.
Select one master and multiple workers and launch applications on each.

## Json settings
To run UCL, two Json files are required.
* virtual-screen-def.json: execution environment such as display and node(SoCs/VMs/PCs) informations.
* app.json: application execution information such as target node and application execution command.

Json files need to be created correctly for your execution environment.

## Workers side
Before running master API, the worker side needs to launch ucl-launcher.
ucl-launcher needs the path to virtual-screen-def.json, so please set it with "-f" option. (default path is /etc/uhmi-framework/virtual-screen-def.json)

**Note:** Master node may also work as worker. 

- Options of ucl-launcher
  - -H: search ucl-launchar param by hostname from VScrnDef file
  - -N: search ucl-launchar param by node id from VScrnDef file (default -1)
  - -d: verbose debug log
  - -f: virtual-screen-def.json file Path (default "/etc/uhmi-framework/virtual-screen-def.json")
  - -v: verbose info log (default true)

```
ucl-launcher -f <path to virtual-screen-def.json>
```

## Master side
After launch all workers, launch ucl-distrib-com in master terminal.
This ucl-distrib-com needs virtual-screen-def.json and app.json, so please set virtual-screen-def.json with argument and set app.json with iostream.

- Options of ucl-distrib-com
  - -d: verbose debug log
  - -f: force the execution of the application even if some nodes are not alive.
  - -ip: IP address of the device executing this command
  - -n: node information (Used when virtual-screen-def.json is not given as an argument, e.g. -n "hostname":"ip":"port")
  - -v: verbose info log (default true)

```
cat <path to app.json> | ucl-distrib-com <path to virtual-screen-def.json>
```

# Colaborate with RVGPU
UCL can be combined with RVGPU in our repository to display applications remotly.
rvgpu-wlproxy enables to remotely display wayland client applications using RVGPU.
UCL has ucl-virtio-gpu-wl-send/recv, which can easily execute and effectively utilize RVGPU and rvgpu-wlproxy.

## How to install RVGPU
For instructions on how to install remove-virtio-gpu, please refer to the [README](https://github.com/unified-hmi/remote-virtio-gpu).

## How to install rvgpu-wlproxy
For instructions on how to install rvgpu-wlproxy, please refer to the [README](https://github.com/unified-hmi/rvgpu-wlproxy).

## How to run UCL
By launching ucl-virtio-gpu-wl-send on RVGPU client side and ucl-virtio-gpu-wl-recv on RVGPU server side, it can be executed without detailed configuration.
Additionally, by using the app.json, remote rendering of the application across multiple nodes can be achieved easily.

**Note:** ucl-virtio-gpu-wl-send/recv only support RVGPU version v1.1.0 or higher.

### Run ucl-virtio-gpu-wl-recv
ucl-virtio-gpu-wl-recv  optionally receives information about RVGPU server and executes the rvgpu-renderer.

- Options
  - -B: remote-virtio-gpu reciever color config (default "0x33333333")
  - -P: specify remote-virtio-gpu reciever port (default "55667")
  - -S: specify remote-virtio-gpu reciever surface ID (default "9000")
  - -d: verbose debug log
  - -s: remote-virtio-gpu reciever screen config (default "1920x1080@0,0")

```
ucl-virtio-gpu-wl-recv -s 1920x1080@0,0 -P 55667
```

### Run ucl-virtio-gpu-wl-send
ucl-virtio-gpu-wl-send optionally receives information about RVGPU client and executes rvgpu-proxy and rvgpu-wlproxy to rendere Wayland client applications remotely.

- Options
  - -appli_name: specify application name (default "ucl-virtio-gpu-wl-send")
  - -d: verbose debug log
  - -n: Specify multiple -n options (default 127.0.0.1:55667)
  - -s: remote-virtio-gpu sender screen config (default "1920x1080@0,0")
  - -v: verbose info log (default true)

```
sudo -i
modprobe virtio-gpu
modprobe virtio-lo
ucl-virtio-gpu-wl-send -s 1920x1080@0,0 -n 127.0.0.1:55667 glmark2-es2-wayland
```

### Using Json configuration
To launch RVGPU and rvgpu-wlproxy from UCL, select ucl-virtio-gpu-wl-send/recv as the application to be launched by UCL in app.json.

#### Json settings
Sample Json files are located in the "$GOPATH/ucl-tools/example" directory.
Please modify Json files according to your own execution environment referring to samples.

#### Workers side
Execute the UCL command for Worker on the all hosts on which you want to run the application.
```
sudo -i
modprobe virtio-gpu
modprobe virtio-lo
export GOPATH=<your go work directory>
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
ucl-launcher -f $GOPATH/ucl-tools/example/single-node/virtual-screen-def.json
```

#### Master side
After all Worker commands have been executed, launch Master command on any host.
```
export GOPATH=<your go work directory>
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
cat $GOPATH/ucl-tools/example/simgle-node/app.json | ucl-distrib-com $GOPATH/ucl-tools/example/single-node/virtual-screen-def.json
```
