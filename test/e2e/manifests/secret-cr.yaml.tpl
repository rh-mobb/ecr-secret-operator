apiVersion: ecr.mobb.redhat.com/v1alpha1
kind: Secret
metadata:
  name: e2e-ecr-secret
  namespace: ${TEST_NAMESPACE}
spec:
  generated_secret_name: "e2e-docker-pull-secret"
  ecr_registry: "${ECR_REGISTRY}"
  region: "${AWS_REGION}"
  frequency: "1h"
