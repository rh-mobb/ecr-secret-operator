apiVersion: ecr.mobb.redhat.com/v1alpha1
kind: ArgoHelmRepoSecret
metadata:
  name: e2e-argo-helm-secret
  namespace: ${TEST_NAMESPACE}
spec:
  generated_secret_name: "e2e-helm-repo-secret"
  url: "https://${ECR_REGISTRY}/${ECR_REPO_NAME}"
  region: "${AWS_REGION}"
  frequency: "1h"
