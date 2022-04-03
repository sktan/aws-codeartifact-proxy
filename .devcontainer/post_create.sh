#!/usr/bin/env bash

# Install x86 or ARM version of awscliv2 based on current machine architecture
if [[ "$(uname -m)" == "x86_64" ]]; then
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
else
  curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "/tmp/awscliv2.zip"
fi
unzip /tmp/awscliv2.zip -d /tmp/awscliv2
/tmp/awscliv2/aws/install && rm -rf /tmp/awscliv2*

npm install -g aws-cdk
pip install pre-commit
