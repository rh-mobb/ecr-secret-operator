## Create IAM user and Policy

**Notes: These are sample commands. Please fill in your own resource parameters E.g.ARN**

* Create a user and access key

```
aws iam create-user --user-name ecr-bot
aws create-access-key --user-name ecr-bot
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
aws iam attach-user-policy --policy-arn arn:aws:iam::[ACCOUNT_ID]:policy/ECRLoginPolicy --user-name ecr-bot
```

* Set up a specific ECR repository access

```
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

