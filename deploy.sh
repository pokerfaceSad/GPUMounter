#!/usr/bin/env bash

if [ ! $# == 1 ]; then
  echo "Invalid parameter (must be \"deploy\", \"redeploy\" or \"uninstall\")"
  exit
fi

if [ "$1" = "deploy" ]; then
  kubectl create -f deploy/namespace.yaml
  kubectl create -f deploy/service-account.yaml
  kubectl create -f deploy/cluster-role-binding.yaml
  kubectl create -f deploy/gpu-mounter-workers.yaml
  kubectl create -f deploy/gpu-mounter-master.yaml
  kubectl create -f deploy/gpu-mounter-svc.yaml

elif [ "$1" = "redeploy" ]; then
  kubectl delete -f deploy/namespace.yaml
  kubectl delete -f deploy/service-account.yaml
  kubectl delete -f deploy/cluster-role-binding.yaml
  kubectl delete -f deploy/gpu-mounter-workers.yaml
  kubectl delete -f deploy/gpu-mounter-master.yaml
  kubectl delete -f deploy/gpu-mounter-svc.yaml

  kubectl create -f deploy/namespace.yaml
  kubectl create -f deploy/service-account.yaml
  kubectl create -f deploy/cluster-role-binding.yaml
  kubectl create -f deploy/gpu-mounter-workers.yaml
  kubectl create -f deploy/gpu-mounter-master.yaml
  kubectl create -f deploy/gpu-mounter-svc.yaml

elif [ "$1" = "uninstall" ]; then
  kubectl delete -f deploy/namespace.yaml
  kubectl delete -f deploy/service-account.yaml
  kubectl delete -f deploy/cluster-role-binding.yaml
  kubectl delete -f deploy/gpu-mounter-workers.yaml
  kubectl delete -f deploy/gpu-mounter-master.yaml
  kubectl delete -f deploy/gpu-mounter-svc.yaml

else
  echo "Invalid parameter: $1 (must be \"deploy\", \"redeploy\" or \"uninstall\")"
fi
