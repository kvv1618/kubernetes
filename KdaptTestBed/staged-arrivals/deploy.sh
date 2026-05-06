#!/bin/bash
# Usage: ./deploy.sh [SCENARIO] [DELAY_SECS]
#   SCENARIO: burst | gradual | interleaved | reverse | random (default: gradual)
#   DELAY_SECS: seconds between each apply (default: 15)

set -euo pipefail

SCENARIO="${1:-gradual}"
DELAY="${2:-15}"
NAMESPACE="default"

log()  { echo -e "[$(date +%H:%M:%S)] $*"; }
info() { echo -e "[$(date +%H:%M:%S)] $*"; }
warn() { echo -e "[$(date +%H:%M:%S)] $*"; }

# Gradual: light → heavy, mixed resource types
SCENARIO_gradual=(
  cpu.yaml
  mem.yaml
  heavy-cpu-request.yaml
  heavy-mem-request.yaml
  heavy-cpu-util.yaml
  heavy-mem-util.yaml
)

# Burst: all heavies first, smash the scheduler immediately
SCENARIO_burst=(
  heavy-cpu-util.yaml
  heavy-mem-util.yaml
  heavy-cpu-request.yaml
  heavy-mem-request.yaml
  cpu.yaml
  mem.yaml
)

# Interleaved: alternate CPU and memory pressure to force mixed-node decisions
SCENARIO_interleaved=(
  cpu.yaml
  mem.yaml
  heavy-cpu-request.yaml
  heavy-mem-util.yaml
  heavy-cpu-util.yaml
  heavy-mem-request.yaml
)

# Reverse: heavies first, then lights (tests if scheduler recovers/rebalances)
SCENARIO_reverse=(
  heavy-mem-util.yaml
  heavy-cpu-util.yaml
  heavy-mem-request.yaml
  heavy-cpu-request.yaml
  mem.yaml
  cpu.yaml
)

# Random: shuffle on each run
SCENARIO_random=($(ls ./*.yaml | xargs -n1 basename | shuf))

# --- Pick scenario ---
case "$SCENARIO" in
  gradual)    FILES=("${SCENARIO_gradual[@]}") ;;
  burst)      FILES=("${SCENARIO_burst[@]}") ;;
  interleaved)FILES=("${SCENARIO_interleaved[@]}") ;;
  reverse)    FILES=("${SCENARIO_reverse[@]}") ;;
  random)     FILES=("${SCENARIO_random[@]}") ;;
  *)
    echo "Unknown scenario: $SCENARIO"
    echo "Available: burst | gradual | interleaved | reverse | random"
    exit 1
    ;;
esac

cleanup() {
  warn "Cleaning up staged-arrival workloads..."
  for f in ./*.yaml; do
    kubectl delete -f "$f" --ignore-not-found --grace-period=0 2>/dev/null || true
  done
  log "Cleanup complete."
}

cleanup

sleep 20 # Wait for cluster to stabilize after cleanup

snapshot() {
  info "--- Cluster snapshot ---"
  kubectl top nodes 2>/dev/null || true
  echo
}

log "Scenario: $SCENARIO | Delay: ${DELAY}s between arrivals"
echo "Files in order:"
for f in "${FILES[@]}"; do echo "  $f"; done
echo

for YAML in "${FILES[@]}"; do
  FULL_PATH="./$YAML"
  if [[ ! -f "$FULL_PATH" ]]; then
    warn "File not found, skipping: $YAML"
    continue
  fi

  log "Applying: $YAML"
  kubectl apply -f "$FULL_PATH"

  info "Waiting ${DELAY}s before next arrival..."
  sleep "$DELAY"
  snapshot
done

info "All workloads applied. Monitoring for 60s to observe scheduling decisions..."
sleep 60

log "Final state:"
snapshot
