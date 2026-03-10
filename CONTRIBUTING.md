# Contributing to ECR Secret Operator

Thank you for your interest in contributing. This guide covers everything you need to build, test, and release the operator.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Repository Layout](#repository-layout)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Running Tests](#running-tests)
- [Building the Operator Image](#building-the-operator-image)
- [Deploying to OpenShift for Manual Testing](#deploying-to-openshift-for-manual-testing)
- [Testing via OLM](#testing-via-olm)
- [Releasing to OperatorHub](#releasing-to-operatorhub)
- [GitHub Actions Workflows](#github-actions-workflows)
- [Commit and PR Guidelines](#commit-and-pr-guidelines)

---

## Prerequisites

| Tool | Minimum Version | Install |
|---|---|---|
| Go | 1.19 | https://go.dev/dl |
| Podman | 4.x | https://podman.io/docs/installation |
| kubectl / oc | any | https://docs.openshift.com/container-platform/latest/cli_reference/openshift_cli/getting-started-cli.html |
| operator-sdk | v1.28+ | https://sdk.operatorframework.io/docs/installation |
| GNU Make | any |  |

You also need:

- An OpenShift cluster configured with [STS/OIDC](https://docs.openshift.com/container-platform/4.16/authentication/managing_cloud_provider_credentials/cco-mode-sts.html) for manual and OLM testing. It is recommended to test this on a ROSA with HCP cluster.
- An AWS IAM role with `ecr:GetAuthorizationToken` permission, configured to trust the cluster's OIDC provider. See [docs/iam_assume_role.md](docs/iam_assume_role.md) for setup instructions.

---

## Repository Layout

```
.
├── api/v1alpha1/          # CRD type definitions (Secret, ArgoHelmRepoSecret)
├── bundle/                # Generated OLM bundle (do not edit manually)
│   ├── manifests/         # CSV, CRDs, RBAC manifests
│   └── metadata/          # OLM channel and version annotations
├── bundle.Dockerfile      # Dockerfile used to build the bundle image
├── config/                # Kustomize bases and overlays
│   ├── default/           # Default deployment overlay (includes kube-rbac-proxy)
│   ├── manager/           # Operator Deployment definition
│   ├── manifests/bases/   # Base CSV template (source of truth for make bundle)
│   ├── rbac/              # RBAC roles and bindings
│   └── samples/           # Sample CR manifests
├── controllers/           # Reconciliation logic
├── docs/                  # IAM setup guides
├── ecr/                   # AWS ECR token generation
├── samples/               # End-user sample CRs
├── .github/workflows/     # GitHub Actions CI and release workflows
├── Dockerfile             # Multi-arch operator image build
├── Makefile               # All build, test, and release targets
└── PROJECT                # Operator-SDK / Kubebuilder project metadata
```

---

## Development Setup

```bash
git clone git@github.com:rh-mobb/ecr-secret-operator.git
cd ecr-secret-operator

# Download all Go dependencies
go mod download

# Install local build tools (controller-gen, kustomize, setup-envtest)
make controller-gen kustomize envtest
```

---

## Making Changes

### CRD API types

Type definitions live in `api/v1alpha1/`. After editing them, regenerate CRD manifests and DeepCopy methods:

```bash
make manifests generate
```

### Controller logic

Controllers live in `controllers/`. No code generation is required after editing them — just run the tests.

### RBAC

RBAC rules are derived from `// +kubebuilder:rbac` markers in the controller source files. After changing markers, regenerate:

```bash
make manifests
```

---

## Running Tests

The test suite uses [Ginkgo](https://onsi.github.io/ginkgo/) with [envtest](https://book.kubebuilder.io/reference/envtest.html), which runs a real `kube-apiserver` locally — no cluster required.

```bash
make test
```

This runs `manifests`, `generate`, `fmt`, `vet`, and then the full test suite. Coverage output is written to `cover.out`.

To view coverage in a browser:

```bash
go tool cover -html=cover.out
```

---

## Building the Operator Image

The operator image is built as a multi-architecture manifest list (`linux/amd64` and `linux/arm64`) using Podman.

```bash
# Log in to the registry first
podman login quay.io

# Build and push (runs make test first)
make podman-build podman-push IMG=quay.io/rh-mobb/ecr-secret-operator:v<VERSION>
```

To build without pushing (useful for local inspection):

```bash
podman build --platform linux/amd64,linux/arm64 --manifest ecr-secret-operator:dev .
```

---

## Deploying to OpenShift for Manual Testing

### 1. Create the namespace and AWS credentials secret

The operator authenticates to AWS using STS/IRSA (IAM Roles for Service Accounts). The credentials secret must use the `credentials` file format expected by the AWS SDK — not static access keys.

First, follow [docs/iam_assume_role.md](docs/iam_assume_role.md) to create an IAM role that trusts your cluster's OIDC provider. Then:

```bash
oc new-project ecr-secret-operator

# Replace with your IAM role ARN
cat <<EOF > /tmp/credentials
[default]
role_arn = arn:aws:iam::<ACCOUNT_ID>:role/<ROLE_NAME>
web_identity_token_file = /var/run/secrets/openshift/serviceaccount/token
EOF

oc create secret generic aws-ecr-cloud-credentials \
  --from-file=credentials=/tmp/credentials \
  -n ecr-secret-operator
```

### 2. Deploy

```bash
make deploy IMG=quay.io/rh-mobb/ecr-secret-operator:v<VERSION>
```

### 3. Verify

```bash
oc get pods -n ecr-secret-operator
oc logs -n ecr-secret-operator -l control-plane=controller-manager -c manager -f
```

### 4. Create a test CR

```yaml
apiVersion: ecr.mobb.redhat.com/v1alpha1
kind: Secret
metadata:
  name: ecr-secret
  namespace: test-ecr-secret-operator
spec:
  generated_secret_name: ecr-credentials
  ecr_registry: <ACCOUNT_ID>.dkr.ecr.<REGION>.amazonaws.com
  frequency: 10h
  region: <REGION>
```

```bash
oc new-project test-ecr-secret-operator
oc apply -f test-cr.yaml
```

### 5. Confirm reconciliation

```bash
# CR status should show Phase: Updated
oc get secret.ecr.mobb.redhat.com ecr-secret \
  -n test-ecr-secret-operator \
  -o jsonpath='{.status}' | jq

# The generated secret should contain a valid ECR token
oc get secret ecr-credentials -n test-ecr-secret-operator \
  -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d | jq
```

### 6. Tear down

```bash
make undeploy
oc delete project test-ecr-secret-operator ecr-secret-operator
```

---

## Testing via OLM

This validates the operator exactly as OperatorHub would install it.

### 1. Generate and push the bundle image

```bash
make bundle IMG=quay.io/rh-mobb/ecr-secret-operator:v<VERSION>

make bundle-build bundle-push \
  BUNDLE_IMG=quay.io/rh-mobb/ecr-secret-operator-bundle:v<VERSION>
```

Ensure the bundle image is publicly readable on quay.io before proceeding.

### 2. Install via OLM

```bash
oc new-project ecr-secret-operator

# Replace with your IAM role ARN
cat <<EOF > /tmp/credentials
[default]
role_arn = arn:aws:iam::<ACCOUNT_ID>:role/<ROLE_NAME>
web_identity_token_file = /var/run/secrets/openshift/serviceaccount/token
EOF

oc create secret generic aws-ecr-cloud-credentials \
  --from-file=credentials=/tmp/credentials \
  -n ecr-secret-operator

operator-sdk run bundle quay.io/rh-mobb/ecr-secret-operator-bundle:v<VERSION> \
  --namespace ecr-secret-operator
```

### 3. Verify the CSV reached Succeeded

```bash
oc get csv -n ecr-secret-operator -w
```

### 4. Run through the manual test steps above (Create a test CR onwards)

### 5. Tear down

```bash
operator-sdk cleanup ecr-secret-operator --namespace ecr-secret-operator
oc delete project test-ecr-secret-operator ecr-secret-operator
```

---

## Releasing to OperatorHub

OperatorHub publishing is **fully automated** via the [`publish-operatorhub`](.github/workflows/publish-operatorhub.yaml) GitHub Actions workflow. Publishing to both the OpenShift OperatorHub and the public OperatorHub is triggered by creating a GitHub Release — no manual bundle copying or PR opening is required.

### Pre-release checklist

Before creating the GitHub Release, complete the following steps locally:

#### 1. Update the version and changelog

Edit `Makefile` and bump:

```makefile
VERSION ?= <NEW_VERSION>
```

Update `config/manifests/bases/ecr-secret-operator.clusterserviceversion.yaml`:

- Set `containerImage: quay.io/rh-mobb/ecr-secret-operator:v<NEW_VERSION>`
- Set `createdAt` to today's date

Add a new entry to `CHANGELOG.md` for the new version.

#### 2. Run the test suite

```bash
make test
```

#### 3. Regenerate and validate the bundle locally

```bash
make bundle IMG=quay.io/rh-mobb/ecr-secret-operator:v<NEW_VERSION>
```

> **Important:** `make bundle` regenerates `bundle/metadata/annotations.yaml` but does not preserve custom annotations. Verify that the file still contains:
>
> ```yaml
> com.redhat.openshift.versions: "v4.16"
> ```
>
> If it was removed, add it back before proceeding.

```bash
operator-sdk bundle validate ./bundle --select-optional suite=operatorframework
operator-sdk bundle validate ./bundle --select-optional name=operatorhub
operator-sdk bundle validate ./bundle --select-optional name=good-practices
```

#### 4. Run the full OLM smoke test on OpenShift

Follow the steps in [Testing via OLM](#testing-via-olm) to confirm the operator installs and reconciles correctly via OLM before publishing.

#### 5. Commit and push to main

```bash
git add -A
git commit -s -m "Release v<NEW_VERSION>"
git push origin main
```

### Creating the GitHub Release (triggers automation)

Once the pre-release checklist is complete, create the GitHub Release:

```bash
gh release create v<NEW_VERSION> \
  --title "v<NEW_VERSION>" \
  --notes-file CHANGELOG.md
```

This triggers the `publish-operatorhub` workflow automatically. The workflow:

1. Builds and pushes the multi-arch operator image to `quay.io/rh-mobb/ecr-secret-operator:v<NEW_VERSION>`
2. Regenerates and validates the OLM bundle
3. Builds and pushes the bundle image to `quay.io/rh-mobb/ecr-secret-operator-bundle:v<NEW_VERSION>`
4. Opens a PR to [`redhat-openshift-ecosystem/community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod) (OpenShift OperatorHub — FBC format)
5. Opens a PR to [`k8s-operatorhub/community-operators`](https://github.com/k8s-operatorhub/community-operators) (Public OperatorHub — semver format)

Monitor the workflow run:

```bash
gh run list --workflow=publish-operatorhub.yaml --limit 5
gh run watch  # follow the active run
```

### After the workflow completes

Two PRs will be opened automatically — one to each community repo. Both will run their own CI pipelines. Once the CI checks pass, the PRs require human review and approval by the repo maintainers before merging.

Monitor the PR checks:

```bash
gh pr checks --repo redhat-openshift-ecosystem/community-operators-prod
gh pr checks --repo k8s-operatorhub/community-operators
```

Once both PRs are merged, the new version will appear in OperatorHub within a few hours.

---

## GitHub Actions Workflows

Two workflows live in `.github/workflows/`:

### `ci.yaml` — Continuous Integration

Triggers on every push to `main` and every pull request targeting `main`.

| Step | What it does |
|---|---|
| Checkout | Checks out the repository |
| Set up Go | Installs Go using the version in `go.mod` |
| Install build tools | Runs `make controller-gen kustomize envtest` |
| Run tests | Runs `make test` (includes `manifests`, `generate`, `fmt`, `vet`) |
| Upload coverage | Saves `cover.out` as a workflow artifact |

### `publish-operatorhub.yaml` — Release and Publish

Triggers when a GitHub Release is published.

| Job | Runs on | What it does |
|---|---|---|
| `build-and-push` | `ubuntu-latest` | Builds multi-arch operator image, generates and validates OLM bundle, builds and pushes bundle image, uploads bundle as artifact |
| `publish-openshift-operatorhub` | `ubuntu-latest` | Copies bundle into the FBC-based `community-operators-prod` fork, updates `catalog-templates/basic.yaml` and `release-config.yaml` with the new version's upgrade graph, opens PR |
| `publish-public-operatorhub` | `ubuntu-latest` | Copies bundle into the semver-based `community-operators` fork, opens PR |

### Required repository secrets

Configure these under **Settings → Secrets and variables → Actions**:

| Secret | Description |
|---|---|
| `QUAY_USERNAME` | Quay.io robot account username (e.g. `rh-mobb+robot`) |
| `QUAY_PASSWORD` | Quay.io robot account token |
| `OPERATORHUB_PR_TOKEN` | GitHub Personal Access Token (classic) with `public_repo` scope, belonging to an account that has forked both community repos under the `rh-mobb` org |

### Required forks

The `publish-operatorhub` workflow pushes to forks before opening upstream PRs. Ensure these forks exist under the `rh-mobb` org:

```bash
gh repo fork redhat-openshift-ecosystem/community-operators-prod \
  --org rh-mobb --clone=false

gh repo fork k8s-operatorhub/community-operators \
  --org rh-mobb --clone=false
```

---

## Commit and PR Guidelines

- All commits must be signed off with `-s` (`git commit -s`) to satisfy the DCO requirement used by the upstream OperatorHub repos.
- Each PR to the community operator repos must contain exactly **one** commit. Use `git rebase -i origin/main` to squash before pushing.
- Branch naming: `bug/<issue-number>` for bug fixes, `feat/<description>` for features, `release/v<version>` for releases.
- Run `make test` before opening any PR against this repository.
