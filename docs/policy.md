## Create IAM user and Policy

**Notes: These are sample commands. Please fill in your own resource parameters E.g.ARN**

* Create a user and access key

```
aws iam create-user --user-name ecr-bot
aws iam create-access-key --user-name ecr-bot
```

**Notes: Save access key id and key for later usage**

* Create the IAM policy and attach to the user

```
cat <<EOF > /tmp/iam_policy.json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecr:GetAuthorizationToken",
                "ecr:BatchCheckLayerAvailability",
                "ecr:GetDownloadUrlForLayer",
                "ecr:GetRepositoryPolicy",
                "ecr:DescribeRepositories",
                "ecr:ListImages",
                "ecr:DescribeImages",
                "ecr:BatchGetImage",
                "ecr:GetLifecyclePolicy",
                "ecr:GetLifecyclePolicyPreview",
                "ecr:ListTagsForResource",
                "ecr:DescribeImageScanFindings",
                "ecr:InitiateLayerUpload",
                "ecr:UploadLayerPart",
                "ecr:CompleteLayerUpload",
                "ecr:PutImage"
            ],
            "Resource": "*"
        }
    ]
}
EOF
aws iam create-policy \
    --policy-name ECRLoginPolicy \
    --policy-document file:///tmp/iam_policy.json
aws iam attach-user-policy --policy-arn arn:aws:iam::[ACCOUNT_ID]:policy/ECRLoginPolicy --user-name ecr-bot
```
