# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.1] — 2026-03-10

### Security

- **Updated `golang.org/x/net` from `v0.8.0` to `v0.38.0`.** Resolves multiple CVEs in the `x/net` package, including [CVE-2024-45338](https://pkg.go.dev/vuln/GO-2024-3333) (denial of service via HTML parsing). ([#28](https://github.com/rh-mobb/ecr-secret-operator/pull/28))
- **Updated `golang.org/x/oauth2` and `google.golang.org/protobuf`.** Routine dependency security maintenance. ([#29](https://github.com/rh-mobb/ecr-secret-operator/pull/29), [#30](https://github.com/rh-mobb/ecr-secret-operator/pull/30))

### Bug Fixes

- **Fixed builder image incompatibility with Go module version format.** The Dockerfile was using `golang:1.19` as the builder, which does not understand the three-part version directive (`go 1.23.0`) introduced in Go 1.21. The builder image has been updated to `golang:1.23` to match the `go` directive in `go.mod`.

### OLM / OperatorHub

- Updated `com.redhat.openshift.versions` annotation format in `bundle/metadata/annotations.yaml` and `bundle.Dockerfile` to the correct `v4.16` format.
- Updated operator description and maintainer contact to reflect the Red Hat Global Cloud Services SSA Team.

### Changes

- **Added contribution automation workflows.** GitHub Actions workflows for CI, OperatorHub publishing, OpenShift version management, and fork syncing have been added. ([#26](https://github.com/rh-mobb/ecr-secret-operator/pull/26))

---

## [0.5.0] — 2026-03-05

### Bug Fixes

- **Fixed non-stop reconciliation loop.** The controller was unconditionally calling `client.Update` on every reconcile cycle, even immediately after creating a new secret. The update logic is now only executed when the secret already exists. ([#25](https://github.com/rh-mobb/ecr-secret-operator/pull/25))
- **Fixed CR status never reaching `Updated` on first reconcile.** `Status.Phase` was only being set inside the update code path, so a freshly created secret would never have its status reflect success. The status is now set correctly after both create and update operations. ([#25](https://github.com/rh-mobb/ecr-secret-operator/pull/25))

### Changes

- **Replaced deprecated `gcr.io/kubebuilder/kube-rbac-proxy` image.** The `kube-rbac-proxy` sidecar image has been migrated from the deprecated Google Container Registry (`gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0`) to the canonical upstream image on Quay.io, pinned by digest for supply-chain security: `quay.io/brancz/kube-rbac-proxy:v0.21.0@sha256:059a43ab03f230eedb7dde2eb0d910a41e9d7ee04efff0f5cdc7ee614111397a`. ([#23](https://github.com/rh-mobb/ecr-secret-operator/pull/23))
- **Renamed default generated secret from `ecr-docker-secret` to `ecr-credentials`.** All sample manifests, documentation, and the OLM CSV have been updated to use `ecr-credentials` as the example `generated_secret_name`. **This is not a breaking change** — the field is user-configured in the CR spec and existing deployments are unaffected. ([#24](https://github.com/rh-mobb/ecr-secret-operator/pull/24))
- **Added `GenerationChangedPredicate` event filter.** Both controllers now ignore status-only updates, preventing spurious reconcile triggers from status writes back to the API server. ([#25](https://github.com/rh-mobb/ecr-secret-operator/pull/25))
- **Build system migrated from Docker to Podman with multi-architecture support.** The Makefile now uses Podman and builds manifest lists targeting both `linux/amd64` and `linux/arm64`. The operator image base is now `quay.io/rh-mobb/ecr-secret-operator`.
- **Added Apache 2.0 license.** ([#21](https://github.com/rh-mobb/ecr-secret-operator/pull/21))

### OLM / OperatorHub

- Minimum Kubernetes version set to `1.29.0` (OpenShift 4.16+).
- Upgrade graph covers all previously published versions (`v0.1.1` through `v0.4.1`) via `replaces` and `skips` fields.
- Default channel set to `alpha`.

### Upgrade Notes

Users upgrading from any prior version (`v0.1.1` – `v0.4.1`) can upgrade directly to `v0.5.0` — no intermediate upgrades are required. Existing CRs and their `generated_secret_name` values are fully preserved.

---

## [0.4.1] — 2023-04-14

### Changes

- Added `--enable-oci` flag when creating ArgoCD Helm repo secrets to support OCI-based Helm registries.
- Documentation updates to README.

---

## [0.4.0] — 2023-04-03

### Features

- **Added `ArgoHelmRepoSecret` CRD.** New custom resource to automatically manage ArgoCD Helm repository credentials backed by ECR tokens.
- Upgraded Go module and Go version dependencies. ([#12](https://github.com/rh-mobb/ecr-secret-operator/pull/12))

### Changes

- Documentation updates for README and IAM setup guides. ([#11](https://github.com/rh-mobb/ecr-secret-operator/pull/11))

---

## [0.3.2] — 2022-08-15

### Bug Fixes

- **Fixed hardcoded AWS region.** The region is now read from the CR spec rather than being hardcoded in the controller. ([#10](https://github.com/rh-mobb/ecr-secret-operator/pull/10))

### Changes

- Updated OLM bundle manifests.
- Added controller test coverage for region handling.

---

## [0.3.1] — 2022-05-09

### Changes

- Made the AWS credentials secret optional to support workload identity / IRSA configurations.
- Fixed errors related to `ClusterVersion` resource access.
- Updated OLM bundle manifests to v0.3.1.

---

## [0.3.0] — 2022-05-08

### Features

- **Added STS / Assume Role support.** AWS authentication logic has been moved out of the operator binary and into a mounted credentials file, enabling STS `AssumeRoleWithWebIdentity` (IRSA) configurations. ([#5](https://github.com/rh-mobb/ecr-secret-operator/pull/5))
- Updated OLM bundle manifests to v0.3.0.

---

## [0.2.1] — 2022-05-03

### Changes

- Added operator description metadata.

---

## [0.2.0] — 2022-05-03

### Features

- **Added multi-namespace support.** The operator can now be configured to run in `OwnNamespace`, `SingleNamespace`, `MultiNamespace`, or `AllNamespaces` install modes. ([#4](https://github.com/rh-mobb/ecr-secret-operator/pull/4))
- Updated OLM bundle to v2.0 media type.

---

## [0.1.1] — 2022-04-28

### Features

- Initial public release of the ECR Secret Operator.
- Automatically refreshes AWS ECR authentication tokens as Kubernetes `kubernetes.io/dockerconfigjson` secrets.
- Configurable refresh frequency via the `Secret` CRD `spec.frequency` field.
- OLM bundle and OperatorHub metadata included.
