# FAQ

### Q: `mknod: not found` Or `Failed to execute cmd: mknod`
A: You need to make sure `mknod` is available in your image/container.

### Q: How to set CGroup Driver?
A: CGroup Driver can be set in [/deploy/gpu-mounter-workers.yaml](https://github.com/pokerfaceSad/GPUMounter/blob/163ef7b10e7b53180033d1585c9e637c72b3b105/deploy/gpu-mounter-workers.yaml) by environment variable `CGROUP_DRIVER`(default: cgroupfs).

