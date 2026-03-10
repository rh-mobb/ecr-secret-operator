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
├── .github/workflows/
│   ├── ci.yaml                    # PR and push: test, build image, build+validate bundle
│   ├── publish-operatorhub.yaml   # Release: build, push, open OperatorHub PRs
│   ├── sync-forks.yaml            # Reusable: sync community-operators forks with upstream
│   └── add-openshift-version.yaml # Daily: detect new OCP versions, open catalog PR
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

OLM testing can be done against either a **PR build** (before merging) or a **release build** (pre-release smoke test). Both approaches use `operator-sdk run bundle` to install the operator exactly as OperatorHub would.

### Testing a PR build

Every pull request automatically builds and pushes tagged images to quay.io via the CI workflow:

| Image | Tag format | Example |
|---|---|---|
| Operator image | `pr-<number>` | `quay.io/rh-mobb/ecr-secret-operator:pr-42` |
| Bundle image | `pr-<number>` | `quay.io/rh-mobb/ecr-secret-operator-bundle:pr-42` |

To find the PR number, check the pull request URL or run:

```bash
gh pr list
```

Ensure the CI workflow has completed and both images are publicly readable on quay.io before proceeding.

#### 1. Create the namespace and AWS credentials secret

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

#### 2. Install the PR build via OLM

```bash
PR=<PR_NUMBER>

operator-sdk run bundle \
  quay.io/rh-mobb/ecr-secret-operator-bundle:pr-${PR} \
  --namespace ecr-secret-operator
```

#### 3. Verify the CSV reached Succeeded

```bash
oc get csv -n ecr-secret-operator -w
```

#### 4. Create a test CR and confirm reconciliation

```bash
oc new-project test-ecr-secret-operator
```

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
oc apply -f test-cr.yaml

# CR status should show Phase: Updated
oc get secret.ecr.mobb.redhat.com ecr-secret \
  -n test-ecr-secret-operator \
  -o jsonpath='{.status}' | jq

# The generated secret should contain a valid ECR token
oc get secret ecr-credentials -n test-ecr-secret-operator \
  -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d | jq
```

#### 5. Tear down

```bash
operator-sdk cleanup ecr-secret-operator --namespace ecr-secret-operator
oc delete project test-ecr-secret-operator ecr-secret-operator
```

---

### Testing a release build

Use this flow as the final pre-release smoke test before creating the GitHub Release.

#### 1. Generate and push the bundle image

```bash
make bundle IMG=quay.io/rh-mobb/ecr-secret-operator:v<VERSION>

make bundle-build bundle-push \
  BUNDLE_IMG=quay.io/rh-mobb/ecr-secret-operator-bundle:v<VERSION>
```

Ensure the bundle image is publicly readable on quay.io before proceeding.

#### 2. Install via OLM

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

#### 3. Verify the CSV reached Succeeded

```bash
oc get csv -n ecr-secret-operator -w
```

#### 4. Run through the Create a test CR and confirm reconciliation steps above

#### 5. Tear down

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

Four workflows live in `.github/workflows/`:

### `ci.yaml` — Continuous Integration

Triggers on every push to `main` and every pull request targeting `main`. All three jobs run in parallel.

| Job | What it does |
|---|---|
| `test` | Installs build tools, runs `make test` (includes `manifests`, `generate`, `fmt`, `vet`, and the full Ginkgo/envtest suite), uploads `cover.out` as an artifact |
| `build-image` | Builds the multi-arch (`linux/amd64` + `linux/arm64`) operator image using Podman. On PRs, pushes the image to `quay.io/rh-mobb/ecr-secret-operator:pr-<number>` so it can be pulled for manual OLM testing |
| `build-bundle` | Installs `operator-sdk`, runs `make bundle`, restores the `com.redhat.openshift.versions` annotation, validates the bundle against the `operatorframework`, `operatorhub`, and `good-practices` validator suites, then builds the bundle image. On PRs, pushes it to `quay.io/rh-mobb/ecr-secret-operator-bundle:pr-<number>` |

All three jobs must pass before a PR can be merged.

PR image tags are ephemeral — they are intended for manual testing only and will be overwritten by subsequent PRs with the same number. Production images are only published by the `publish-operatorhub.yaml` workflow on release.

### `sync-forks.yaml` — Fork Sync

A reusable workflow (`workflow_call`) invoked automatically by `publish-operatorhub.yaml` before any PR is opened. Can also be triggered manually via `workflow_dispatch` if needed.

```bash
# Trigger manually if needed before a release
gh workflow run sync-forks.yaml
```

Uses `gh repo sync` to keep both forks' `main` branches as exact mirrors of upstream. If a fork has diverged with commits not in upstream, the sync will refuse to overwrite — add `--force` to the command in that case.

### `publish-operatorhub.yaml` — Release and Publish

Triggers when a GitHub Release is published. Job execution order:

```
release published
      │
      ├── sync-forks (reusable)     ← syncs both forks in parallel
      │       ├── sync community-operators-prod
      │       └── sync community-operators
      │
      ├── build-and-push            ← builds images, generates bundle
      │
      └── (both of the above complete)
              │
              ├── publish-openshift-operatorhub  ← FBC PR to community-operators-prod
              └── publish-public-operatorhub     ← semver PR to community-operators
```

| Job | What it does |
|---|---|
| `sync-forks` | Calls `sync-forks.yaml` to reset both fork `main` branches to upstream before any changes are made |
| `build-and-push` | Builds multi-arch operator image, generates and validates OLM bundle, builds and pushes bundle image, uploads bundle as artifact |
| `publish-openshift-operatorhub` | Copies bundle into the FBC-based `community-operators-prod` fork, updates `catalog-templates/basic.yaml` and `release-config.yaml`, opens PR |
| `publish-public-operatorhub` | Copies bundle into the semver-based `community-operators` fork, opens PR |

### `add-openshift-version.yaml` — New OpenShift Version Detection

Runs daily at 06:00 UTC and can also be triggered manually. Detects when Red Hat adds a new OpenShift minor version catalog directory (e.g. `v4.22`) to `community-operators-prod` that isn't yet listed in this operator's `ci.yaml` `catalog_mapping`, and automatically opens a PR to add it.

The workflow:
1. Syncs the `rh-mobb/community-operators-prod` fork with upstream
2. Lists all `vX.Y` catalog directories present in the upstream `catalogs/` directory
3. Compares them against the versions already in `operators/ecr-secret-operator/ci.yaml`
4. For any new version at or above `v4.16` (the operator's `minKubeVersion`), adds it to the `basic.yaml` catalog mapping
5. Opens a PR to `community-operators-prod` with a checklist reminding the reviewer to confirm the operator has been tested on the new version

```bash
# Trigger manually if you want to check for new versions immediately
gh workflow run add-openshift-version.yaml
```

> **Note:** The PR opened by this workflow adds the new version to the FBC catalog mapping only. It does **not** create a new operator release — the existing latest bundle will be made available on the new OpenShift version. If the new OpenShift version requires code changes (e.g. due to API deprecations), create a full release instead.

### Required repository secrets

Configure these under **Settings → Secrets and variables → Actions**:

| Secret | Description |
|---|---|
| `QUAY_USERNAME` | Quay.io robot account username (e.g. `rh-mobb+robot`) |
| `QUAY_PASSWORD` | Quay.io robot account token |
| `OPERATORHUB_PR_TOKEN` | Fine-grained Personal Access Token belonging to `rh-mobb-bot` with **Contents: read/write**, **Pull requests: read/write**, and **Workflows: read/write** on `rh-mobb/ecr-secret-operator`, `rh-mobb-bot/community-operators-prod`, and `rh-mobb-bot/community-operators` |

### Required bot account and forks

The workflows use a dedicated bot account (`rh-mobb-bot`) rather than a personal account so that automation is not tied to any individual's credentials. The token must cover three repositories with the following permissions:

- `rh-mobb-bot/community-operators-prod` and `rh-mobb-bot/community-operators`: **Contents** read/write, **Pull requests** read/write
- `rh-mobb/ecr-secret-operator`: **Contents** read/write, **Pull requests** read/write, **Workflows** read/write

---

## Commit and PR Guidelines

- All commits must be signed off with `-s` (`git commit -s`) to satisfy the DCO requirement used by the upstream OperatorHub repos.
- Each PR to the community operator repos must contain exactly **one** commit. Use `git rebase -i origin/main` to squash before pushing.
- Branch naming: `bug/<issue-number>` for bug fixes, `feat/<description>` for features, `release/v<version>` for releases.
- Run `make test` before opening any PR against this repository.
