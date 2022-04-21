# ECR Secret Operator

Amazon Elastic Container Registry [Private Registry Authentication](https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry_auth.html) provides a temporary token that is valid only for 12 hours. It is a challenge for automatic container image build process to refresh the token or secret in a timely manner.

This operators frequently talks with AWS ECR GetAuthroization Token and create/update the secret, so that the service account can perform docker image build.


## How to use this operator

### Prerequisites

* [Create an ECR private repository](https://docs.aws.amazon.com/AmazonECR/latest/userguide/repository-create.html)
* Create an IAM user with appropriate IAM Policy and ECR registry Policy. [Example](./Docs/policy.md)
* Create An Openshift Cluster
* Install [Operator SDK CLI](https://sdk.operatorframework.io/docs/installation/)

### Install the operator

```
oc new-project ecr-secret-operator
operator-sdk run bundle quay.io/mobb/ecr-secret-operator-bundle:v0.1.1
```

![Installed Operator](./docs/images/operator.png)

### Create the AWS IAM Secret 

```
oc new-project test-ecr-secret-operator
oc create secret generic ecr-iam-secret --from-literal aws_secret_access_id=[ID] --from-literal aws_secret_access_key=[KEY] --from-literal region=us-east-2
```

### Create the ECR Secret CRD

```
apiVersion: ecr.mobb.redhat.com/v1alpha1
kind: Secret
metadata:
  name: ecr-secret
  namespace: test-ecr-secret-operator
spec:
  generated_secret_name: ecr-docker-secret
  ecr_registry: [ACCOUNT_ID].dkr.ecr.us-east-2.amazonaws.com
  frequency: 10h
  aws_iam_secret:
    name: ecr-iam-secret
```

```
oc create -f samples/ecr_v1alpha1_secret.yaml
```

A docker registry secret is created by the operator momentally and the token is patched every 10 hours

```
oc get secret ecr-docker-secret   
NAME                TYPE                             DATA   AGE
ecr-docker-secret   kubernetes.io/dockerconfigjson   1      16h
```

### A sample build process with generated secret


Link the secret to builder

```
oc secrets link builder ecr-docker-secret 
```

Configure [build config](./samples/build-config.yaml) to point to your ECR Container repository

```
oc create imagestream ruby
oc tag openshift/ruby:2.5-ubi8 ruby:2.5
oc create -f samples/build-config.yaml
oc start-build ruby-sample-build --wait
```

Build should succeed and push the image to the the private ECR Container repository

![Success Build](./docs/images/build.png)

