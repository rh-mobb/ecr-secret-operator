apiVersion: ecr.mobb.redhat.com/v1alpha1
kind: Secret
metadata:
  name: e2e-ecr-secret-bad-region
  namespace: ${TEST_NAMESPACE}
spec:
  generated_secret_name: "e2e-docker-pull-secret-bad"
  ecr_registry: "${ECR_REGISTRY}"
  region: "xx-invalid-99"
  frequency: "1h"
