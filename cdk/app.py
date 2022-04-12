#!/usr/bin/env python3
import os

import aws_cdk as cdk

from cdk.code_artifact_proxy import CodeArtifactProxy

app = cdk.App()

cdk_env = cdk.Environment(
    account=os.environ["CDK_DEFAULT_ACCOUNT"],
    region=os.environ["CDK_DEFAULT_REGION"],
)

proxy = CodeArtifactProxy(
    app,
    "codeartifact-proxy",
    # Replace the 3 lines below with your own values
    domain_name="mycodeartifactdomain",
    repository_name="internalrepo",
    vpc_id="vpc-1234567",
    env=cdk_env,
)

# This is actually optional if you do not already have a codeartifact repository
proxy.create_code_artifact()
proxy.create_loadbalanced_fargate()

app.synth()
