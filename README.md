# AWS Code Artifact Proxy

An AWS Code Artifact Proxy that allows you to point your package managers to Code Artifact without the need of managing credentials.

## Why was this built?

Not every user who pulls code from your private codeartifact repository needs AWS credentials:
 - Users of CLI tooling you are deploying internally in your comapny
 - Developers of applications that don't interact with AWS but rely on a private Python / Node library.
 - Maybe you have firewalling requirements or want the ability to see which packages are being installed by your developers?

## Features:

Although I haven't been able to test them all, the proxy should support the following artifact types (replace `artifacts.example.com` with your deployed proxy hostname).

| Repository Type | Tested | URL                                   |
| --------------- | ------ | ------------------------------------- |
| Pypi            | Yes    | https://artifacts.example.com/simple/ |
| NPM             | Yes    | https://artifacts.example.com/        |
| Maven           | No     | https://artifacts.example.com/        |
| Nuget           | No     | https://artifacts.example.com/        |

Currently we only support choosing a single repository at launch, athough maybe in the future I will look at automatically figure out which repository to use based on the useragent. This should simplify setup.

## How to Use?

There are a variety of options for running `aws-codeartifact-proxy`:

- Download the release from the Github page and run it directly on any Linux server.
- Run the container `sktan/aws-codeartifact-proxy` on any capable host (AWS ECS, AWS EC2, Linux / Windows VM)
  - The [`cdk` directory](./cdk) contains a CDK template for deployment to AWS (requires Python)
- Run as a [Nix flake](https://nixos.wiki/wiki/Flakes):
  ```shell
  nix run github:sktan/aws-codeartifact-proxy
  ```

Configuration is done via Environment Variables:

| Environment Variable  |  Required? | Description             |
| --------------------  | ---------- | ----------------------- |
| `CODEARTIFACT_REPO`   | Yes        | Your CodeArtifact Repository Name (e.g. sandbox) |
| `CODEARTIFACT_DOMAIN` | Yes        | Your CodeArtifact Domain (e.g. sktansandbox) |
| `CODEARTIFACT_TYPE`   | No         | Use one of the following: pypi, npm, maven, nuget |
| `CODEARTIFACT_OWNER`  | No         | The AWS Account ID of the CodeArtifact Owner (if it's your own account, it can be empty) |
| `LISTEN_PORT`         | No         | Port on which the proxy should listen.  Defaults to 8080 |

By default, the proxy will choose to use the Pypi as its type.

Once you have started the proxy with valid AWS credentials (this uses the [default credential provider chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)), you should receive similar output to this:

```
2022/04/03 04:41:53 Authenticating against CodeArtifact
2022/04/03 04:41:53 Authorization successful
2022/04/03 04:41:53 Requests will now be proxied to https://sktansandbox-1234567890.d.codeartifact.ap-southeast-2.amazonaws.com/pypi/sandbox/
```

### Docker Examples

Docker CLI:

```
docker run -v /root/.aws/:/.aws -e AWS_PROFILE=sktansandbox -e CODEARTIFACT_DOMAIN=sktansandbox -e CODEARTIFACT_REPO=sandbox -e CODEARTIFACT_TYPE=npm -p 8080:8080 sktan/aws-codeartifact-proxy
```

Docker Compose:

```yaml
version: '3.1'

services:
  codeartifact-proxy:
    image: sktan/aws-codeartifact-proxy
    restart: always
    volumes:
      - /home/sktan/.aws/:/.aws
    environment:
      AWS_PROFILE: sktansandbox
      CODEARTIFACT_DOMAIN: sktansandbox
      CODEARTIFACT_REPO: sandbox
      CODEARTIFACT_OWNER: 1234567890
      CODEARTIFACT_TYPE: pypi
    ports:
      - 8080:8080
```

### AWS CDK Example

You will be able to use the CDK template in the `cdk` directory to create a Load Balancer, a fargate container and a CodeArtifact repository (if you desire).

Modify the variables in app.py (or copy the `cdk/code_artifact_proxy.py` file to your codebase).

```
root ➜ /workspaces/aws-codeartifact-proxy/cdk (cdk ✗) $ pipenv install
Installing dependencies from Pipfile.lock (1a118d)...
  🐍   ▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉▉ 0/0 — 00:00:00
To activate this project's virtualenv, run pipenv shell.
Alternatively, run a command inside the virtualenv with pipenv run.
root ➜ /workspaces/aws-codeartifact-proxy/cdk (cdk ✗) $ pipenv run cdk deploy
```

If you'd rather use your own CDK codebase, you can use the following snippet in your `app.py` file:

```python
# Replace me with where you have placed your codeartifact module
from cdk.code_artifact_proxy import CodeArtifactProxy

proxy = CodeArtifactProxy(
    app,
    "codeartifact-proxy",
    # Replace the 3 lines below with your own values
    domain_name="mycodeartifactdomain",
    repository_name="internalrepo",
    vpc_id="vpc-1234567",
    env=cdk.Environment(
        account=os.environ["CDK_DEFAULT_ACCOUNT"],
        region=os.environ["CDK_DEFAULT_REGION"],
    ),
)

# This is actually optional if you do not already have a codeartifact repository
proxy.create_code_artifact()
proxy.create_loadbalanced_fargate()
```

## Testing Access

And to test that it is working, using `pip` against the proxy should result in similar output:

```
## CLI Output
root ➜ /workspaces/aws-codeartifact-proxy (master ✗) $ pip download --index-url="http://localhost:8080/simple" --no-deps boto3
Looking in indexes: http://localhost:8080/simple
Collecting boto3
  Downloading http://localhost:8080/simple/boto3/1.21.32/boto3-1.21.32-py3-none-any.whl (132 kB)
     ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 132.4/132.4 KB 20.6 MB/s eta 0:00:00
Saved ./boto3-1.21.32-py3-none-any.whl
Successfully downloaded boto3

## Proxy Output
2022/04/03 04:52:44 REQ: 127.0.0.1:52066 GET "/simple/boto3/" "pip/22.0.4 ...."
2022/04/03 04:52:44 Sending request to https://sktansandbox-1234567890.d.codeartifact.ap-southeast-2.amazonaws.com/pypi/sandbox/simple/boto3/
2022/04/03 04:52:44 RES: 127.0.0.1:52066 "GET" 200 "/simple/boto3/" "pip/22.0.4 ...."
```

NPM output:
```
root ➜ /tmp (master ✗) $ npm  view --registry http://localhost:8080 axios dist.tarball
http://localhost:8080/axios/-/axios-0.26.1.tgz

root ➜ /tmp (master ✗) $ npm install --registry http://localhost:8080 axios

added 2 packages in 2s

1 package is looking for funding
  run `npm fund` for details
```

### IAM Permissions

Use the following permissions to grant the proxy ReadOnly access to the CodeArtifact repository.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "codeartifact:Describe*",
                "codeartifact:Get*",
                "codeartifact:List*",
                "codeartifact:ReadFromRepository"
            ],
            "Effect": "Allow",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "sts:GetServiceBearerToken",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "sts:AWSServiceName": "codeartifact.amazonaws.com"
                }
            }
        }
    ]
}
```

## Contributing

If you'd like to contribute to this project, please feel free to raise a pull request. I would highly recommend using the devcontainer setup in this repo, as it will provide you a working development environment.

If you find any bugs, please raise it as a Github issue and I will have a look at it.
