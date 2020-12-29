# GPU Mounter

![GPUMounter License](https://img.shields.io/github/license/pokerfaceSad/GPUMounter.svg)  ![GPUMounter master CI badge](https://github.com/pokerfaceSad/GPUMounter/workflows/GPUMounter-master%20CI/badge.svg)  ![GPUMounter worker CI badge](https://github.com/pokerfaceSad/GPUMounter/workflows/GPUMounter-worker%20CI/badge.svg)

GPU Mounter is a kubernetes plugin which enables add or remove GPU resources for running Pods. This [Introduction(In Chinese)](https://zhuanlan.zhihu.com/p/338251170) is recommended to read which can help you understand what and why is GPU Mounter.

<div align="center"> <img src="docs/images/SchematicDiagram.png" alt="Schematic Diagram Of GPU Dynamic Mount"  /> </div>

## Features

* Supports add or remove GPU resources of running Pod without stopping or restarting
* Compatible with kubernetes scheduler

## Prerequisite 

* Kubernetes v1.16.2 / v1.18.6 (other version not tested, v1.13+ is required, v1.15+ is recommended)
* Docker 19.03/18.09 (other version not tested)
* Nvidia GPU device plugin
* `nvidia-container-runtime` (must be configured as default runtime)

NOTE: If you are using GPU Mounter on Kubernetes v1.13 or v1.14, you need to [manually enable the feature `KubeletPodResources`](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/). It is enabled by default in Kubernetes v1.15+.

## Deploy

* label GPU nodes with `gpu-mounter-enable=enable`

```shell
kubectl label node <nodename> gpu-mounter-enable=enable
```

* deploy

```bash
./deploy.sh deploy
```

* uninstall

```shell
./deploy.sh uninstall
```

## Quick Start

See [QuickStart.md](docs/guide/QuickStart.md)

## FAQ

See  [FAQ.md](docs/guide/FAQ.md)

## License

This project is licensed under the Apache-2.0 License.

## Issues and Contributing

* Please let me know by [Issues](https://github.com/pokerfaceSad/GPUMounter/issues/new) if you experience any problems
* [Pull requests](https://github.com/pokerfaceSad/GPUMounter/pulls) are very welcomed, if you have any ideas to make GPU Mounter better.
