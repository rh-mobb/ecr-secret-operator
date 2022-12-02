## Create IAM user and Policy

**Notes: These are sample commands. Please fill in your own resource parameters E.g. ARN**

* Create the IAM policy

```bash
cat <<EOF > /tmp/iam_policy.json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecr:GetAuthorizationToken"
            ],
            "Resource": "*"
        }
    ]
}
EOF
aws iam create-policy \
    --policy-name ECRLoginPolicy \
    --policy-document file:///tmp/iam_policy.json
```

* Create an IAM user and access key, then attach it to the IAM policy

```bash
aws iam create-user --user-name ecr-bot
aws create-access-key --user-name ecr-bot
aws iam attach-user-policy --policy-arn arn:aws:iam::[ACCOUNT_ID]:policy/ECRLoginPolicy --user-name ecr-bot
```

**Notes: Save access key id and key for later usage**

* Set up a specific Amazon ECR repository access

```bash
cat <<EOF > /tmp/repo_policy.json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowPushPull",
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "arn:aws:iam::[ACCOUNT_ID]:user/ecr-bot"
                ]
            },
            "Action": [
                "ecr:BatchGetImage",
                "ecr:BatchCheckLayerAvailability",
                "ecr:CompleteLayerUpload",
                "ecr:GetDownloadUrlForLayer",
                "ecr:InitiateLayerUpload",
                "ecr:PutImage",
                "ecr:UploadLayerPart"
            ]
        }
    ]
}
EOF

aws ecr set-repository-policy --repository-name test --policy-text file:///tmp/repo_policy.json
```

* Create a Kubernetes Secret with IAM user

```bash
cat <<EOF > /tmp/credentials
[default]
aws_access_key_id=""
aws_secret_access_key=""
EOF


oc create secret generic aws-ecr-cloud-credentials --from-file=credentials=/tmp/credentials
```
