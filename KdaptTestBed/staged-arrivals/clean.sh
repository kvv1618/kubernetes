#!/bin/bash

set -euo pipefail

log()  { echo -e "[$(date +%H:%M:%S)] $*"; }
info() { echo -e "[$(date +%H:%M:%S)] $*"; }
warn() { echo -e "[$(date +%H:%M:%S)] $*"; }

cleanup() {
  warn "Cleaning up staged-arrival workloads..."
  for f in ./*.yaml; do
    kubectl delete -f "$f" --ignore-not-found --grace-period=0 2>/dev/null || true
  done
  log "Cleanup complete."
}

cleanup

sleep 20 # Wait for cluster to stabilize after cleanup