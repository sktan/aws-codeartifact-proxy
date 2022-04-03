# AWS Code Artifact Proxy

An AWS Code Artifact Proxy that allows you to point your package managers to Code Artifact without the need of managing credentials.
## Why was this built?

Not every user who pulls code from your private codeartifact repository needs AWS credentials:
 - Users of CLI tooling you are deploying internally in your comapny
 - Developers of applications that don't interact with AWS but rely on a private Python / Node library.
 - Maybe you have firewalling requirements or want the ability to see which packages are being installed by your developers?

## Features:

Although I haven't been able to test them all (mostly because I don't use the languages), the proxy should support the following artifact types (replace `artifacts.example.com` with your deployed proxy hostname)

| Repository Type | Tested | URL                                   |
| --------------- | ------ | ------------------------------------- |
| Pypi            | Yes    | https://artifacts.example.com/simple/ |
| NPM             | Yes    | https://artifacts.example.com/        |
| Maven           | No     | https://artifacts.example.com/        |
| Nuget           | No     | https://artifacts.example.com/        |

Currently we only support choosing a single repository at launch, athough maybe in the future I will look at auto-resovling the request and figure out which repository to use based on the useragent. This should simplify setup.

## How to Use?

You can run this in three easy ways.

1. Download the release from the Github page, and run it on any linux server.
2. Use the container `sktan/aws-codeartifact-proxy` and run it on any capable host (AWS ECS, AWS EC2, Linux / Windows VM)
3. Use the pre-built CDK template found in the `cdk` directory and deploy it to your environment (requires Python)

By default, the proxy will choose to use the Pypi as it's type. If you would like to use a separate registry type, set your `CODEARTIFACT_TYPE` environment variable to one of the following:
- pypi
- npm
- maven
- nuget

Once you have started the proxy with valid AWS credentials (this uses the [default credential provider chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)), you should receive similar output to this:

```
2022/04/03 04:41:53 Authenticating against CodeArtifact
2022/04/03 04:41:53 Authorization successful
2022/04/03 04:41:53 Requests will now be proxied to https://sktansandbox-1234567890.d.codeartifact.ap-southeast-2.amazonaws.com/pypi/sandbox/
```

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

## Contributing

If you'd like to contribute to this project, please feel free to raise a pull request. I would highly recommend using the devcontainer setup in this repo, as it will provide you a working development environment.

If you find any bugs, please raise it as a Github issue and I will have a look at it.
