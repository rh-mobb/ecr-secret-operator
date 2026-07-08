#!/usr/bin/env bash
set -euo pipefail

: "${ECR_REGISTRY:?ECR_REGISTRY must be set}"
: "${TEST_NAMESPACE:?TEST_NAMESPACE must be set}"
: "${AWS_REGION:=us-east-2}"
export ECR_REPO_NAME="${ECR_REPO_NAME:-ecr-e2e-test-repo}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFESTS_DIR="${SCRIPT_DIR}/../manifests"
PASS=0
FAIL=0
TESTS_RUN=0

log()  { echo "--- [$(date +%H:%M:%S)] $*"; }
pass() { log "PASS: $1"; PASS=$((PASS + 1)); }
fail() { log "FAIL: $1"; FAIL=$((FAIL + 1)); }

render_template() {
  envsubst < "$1"
}

wait_for_secret() {
  local name="$1" timeout="$2"
  log "Waiting up to ${timeout}s for secret/${name}..."
  local end=$((SECONDS + timeout))
  while [ $SECONDS -lt $end ]; do
    if oc get secret "$name" -n "$TEST_NAMESPACE" &>/dev/null; then
      return 0
    fi
    sleep 5
  done
  return 1
}

wait_for_cr_phase() {
  local kind="$1" name="$2" expected="$3" timeout="$4"
  log "Waiting up to ${timeout}s for ${kind}/${name} phase=${expected}..."
  local end=$((SECONDS + timeout))
  while [ $SECONDS -lt $end ]; do
    local phase
    phase=$(oc get "$kind" "$name" -n "$TEST_NAMESPACE" \
              -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "$phase" = "$expected" ]; then
      return 0
    fi
    sleep 5
  done
  return 1
}

cleanup_cr() {
  oc delete "$1" "$2" -n "$TEST_NAMESPACE" --ignore-not-found=true 2>/dev/null || true
}

cleanup_secret() {
  oc delete secret "$1" -n "$TEST_NAMESPACE" --ignore-not-found=true 2>/dev/null || true
}

operator_is_running() {
  local ready
  ready=$(oc get deployment ecr-secret-operator-controller-manager \
            -n "$TEST_NAMESPACE" \
            -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
  [ "${ready:-0}" -ge 1 ]
}

# =========================================================================
# Test 1: Secret CR creates dockerconfigjson secret
# =========================================================================
test_secret_cr_positive() {
  local test_name="Secret CR creates dockerconfigjson secret"
  TESTS_RUN=$((TESTS_RUN + 1))
  log "TEST: ${test_name}"

  cleanup_cr "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret"
  cleanup_secret "e2e-docker-pull-secret"

  render_template "${MANIFESTS_DIR}/secret-cr.yaml.tpl" | oc apply -f -

  if ! wait_for_secret "e2e-docker-pull-secret" 120; then
    fail "${test_name} - generated secret not created within timeout"
    return
  fi

  local secret_type
  secret_type=$(oc get secret "e2e-docker-pull-secret" -n "$TEST_NAMESPACE" \
                  -o jsonpath='{.type}')
  if [ "$secret_type" != "kubernetes.io/dockerconfigjson" ]; then
    fail "${test_name} - expected type kubernetes.io/dockerconfigjson, got ${secret_type}"
    return
  fi

  local data_len
  data_len=$(oc get secret "e2e-docker-pull-secret" -n "$TEST_NAMESPACE" \
               -o jsonpath='{.data.\.dockerconfigjson}' | wc -c)
  if [ "$data_len" -lt 10 ]; then
    fail "${test_name} - .dockerconfigjson data is empty or too short"
    return
  fi

  if ! wait_for_cr_phase "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret" "Updated" 60; then
    fail "${test_name} - CR status.phase never reached 'Updated'"
    return
  fi

  local last_updated
  last_updated=$(oc get secrets.ecr.mobb.redhat.com "e2e-ecr-secret" -n "$TEST_NAMESPACE" \
                   -o jsonpath='{.status.lastUpdatedTime}')
  if [ -z "$last_updated" ]; then
    fail "${test_name} - status.lastUpdatedTime is empty"
    return
  fi

  pass "${test_name}"
}

# =========================================================================
# Test 2: ArgoHelmRepoSecret CR creates repo secret
# =========================================================================
test_argo_helm_cr_positive() {
  local test_name="ArgoHelmRepoSecret CR creates repo secret"
  TESTS_RUN=$((TESTS_RUN + 1))
  log "TEST: ${test_name}"

  cleanup_cr "argohelmreposecrets.ecr.mobb.redhat.com" "e2e-argo-helm-secret"
  cleanup_secret "e2e-helm-repo-secret"

  render_template "${MANIFESTS_DIR}/argo-helm-cr.yaml.tpl" | oc apply -f -

  if ! wait_for_secret "e2e-helm-repo-secret" 120; then
    fail "${test_name} - generated secret not created within timeout"
    return
  fi

  for key in username password url type enableOCI name; do
    local val
    val=$(oc get secret "e2e-helm-repo-secret" -n "$TEST_NAMESPACE" \
            -o jsonpath="{.data.${key}}" 2>/dev/null || echo "")
    if [ -z "$val" ]; then
      fail "${test_name} - missing data key '${key}'"
      return
    fi
  done

  local label
  label=$(oc get secret "e2e-helm-repo-secret" -n "$TEST_NAMESPACE" \
            -o jsonpath='{.metadata.labels.argocd\.argoproj\.io/secret-type}')
  if [ "$label" != "repository" ]; then
    fail "${test_name} - expected argocd label 'repository', got '${label}'"
    return
  fi

  if ! wait_for_cr_phase "argohelmreposecrets.ecr.mobb.redhat.com" \
         "e2e-argo-helm-secret" "Updated" 60; then
    fail "${test_name} - CR status.phase never reached 'Updated'"
    return
  fi

  pass "${test_name}"
}

# =========================================================================
# Test 3: Delete and recreate CR produces new secret
# =========================================================================
test_delete_recreate_cr() {
  local test_name="Delete and recreate Secret CR produces new secret"
  TESTS_RUN=$((TESTS_RUN + 1))
  log "TEST: ${test_name}"

  cleanup_cr "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret"
  cleanup_secret "e2e-docker-pull-secret"
  sleep 5

  render_template "${MANIFESTS_DIR}/secret-cr.yaml.tpl" | oc apply -f -

  if ! wait_for_secret "e2e-docker-pull-secret" 120; then
    fail "${test_name} - secret not created after CR recreation"
    return
  fi

  if ! wait_for_cr_phase "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret" "Updated" 60; then
    fail "${test_name} - CR phase not 'Updated' after recreation"
    return
  fi

  pass "${test_name}"
}

# =========================================================================
# Test 4: Reconciliation recreates deleted secret
# =========================================================================
test_secret_reconciliation() {
  local test_name="Operator recreates deleted secret (reconciliation)"
  TESTS_RUN=$((TESTS_RUN + 1))
  log "TEST: ${test_name}"

  if ! oc get secrets.ecr.mobb.redhat.com "e2e-ecr-secret" -n "$TEST_NAMESPACE" &>/dev/null; then
    render_template "${MANIFESTS_DIR}/secret-cr.yaml.tpl" | oc apply -f -
    wait_for_secret "e2e-docker-pull-secret" 120
  fi

  log "Deleting generated secret to test reconciliation..."
  oc delete secret "e2e-docker-pull-secret" -n "$TEST_NAMESPACE"

  # The controller uses GenerationChangedPredicate, so deleting the generated
  # secret alone won't trigger reconcile. Patch a spec field to bump the
  # CR's generation and force a reconcile cycle.
  log "Patching CR frequency to trigger reconciliation..."
  oc patch secrets.ecr.mobb.redhat.com "e2e-ecr-secret" \
    -n "$TEST_NAMESPACE" \
    --type merge -p '{"spec":{"frequency":"2h"}}'

  if ! wait_for_secret "e2e-docker-pull-secret" 120; then
    fail "${test_name} - secret was not recreated after deletion + reconcile trigger"
    return
  fi

  # Restore original frequency
  oc patch secrets.ecr.mobb.redhat.com "e2e-ecr-secret" \
    -n "$TEST_NAMESPACE" \
    --type merge -p '{"spec":{"frequency":"1h"}}' || true

  pass "${test_name}"
}

# =========================================================================
# Test 5: Invalid region doesn't crash operator
# =========================================================================
test_invalid_region() {
  local test_name="Secret CR with invalid region does not crash operator"
  TESTS_RUN=$((TESTS_RUN + 1))
  log "TEST: ${test_name}"

  cleanup_cr "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret-bad-region"
  cleanup_secret "e2e-docker-pull-secret-bad"

  render_template "${MANIFESTS_DIR}/secret-cr-invalid-region.yaml.tpl" | oc apply -f -

  sleep 30

  if oc get secret "e2e-docker-pull-secret-bad" -n "$TEST_NAMESPACE" &>/dev/null; then
    fail "${test_name} - secret was created despite invalid region"
    cleanup_cr "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret-bad-region"
    cleanup_secret "e2e-docker-pull-secret-bad"
    return
  fi

  if ! operator_is_running; then
    fail "${test_name} - operator crashed after invalid region CR"
    cleanup_cr "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret-bad-region"
    return
  fi

  pass "${test_name}"
  cleanup_cr "secrets.ecr.mobb.redhat.com" "e2e-ecr-secret-bad-region"
}

# =========================================================================
# Test 6: Non-existent registry fails gracefully
# =========================================================================
test_nonexistent_registry() {
  local test_name="Secret CR with non-existent registry fails gracefully"
  TESTS_RUN=$((TESTS_RUN + 1))
  log "TEST: ${test_name}"

  local cr_name="e2e-ecr-secret-bad-registry"
  local secret_name="e2e-docker-pull-secret-bad-reg"
  cleanup_cr "secrets.ecr.mobb.redhat.com" "$cr_name"
  cleanup_secret "$secret_name"

  cat <<EOF | oc apply -f -
apiVersion: ecr.mobb.redhat.com/v1alpha1
kind: Secret
metadata:
  name: ${cr_name}
  namespace: ${TEST_NAMESPACE}
spec:
  generated_secret_name: "${secret_name}"
  ecr_registry: "000000000000.dkr.ecr.${AWS_REGION}.amazonaws.com"
  region: "${AWS_REGION}"
  frequency: "1h"
EOF

  sleep 30

  if ! operator_is_running; then
    fail "${test_name} - operator crashed"
  else
    pass "${test_name}"
  fi

  cleanup_cr "secrets.ecr.mobb.redhat.com" "$cr_name"
  cleanup_secret "$secret_name"
}

# =========================================================================
# Main
# =========================================================================
log "Starting e2e test suite"
log "ECR_REGISTRY=${ECR_REGISTRY}"
log "TEST_NAMESPACE=${TEST_NAMESPACE}"
log "AWS_REGION=${AWS_REGION}"
echo ""

test_secret_cr_positive
test_argo_helm_cr_positive
test_delete_recreate_cr
test_secret_reconciliation
test_invalid_region
test_nonexistent_registry

echo ""
log "=========================================="
log "E2E Test Results: ${TESTS_RUN} run, ${PASS} passed, ${FAIL} failed"
log "=========================================="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
