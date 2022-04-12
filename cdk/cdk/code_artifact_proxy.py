import aws_cdk as cdk
from aws_cdk import (
    Stack,
    aws_ecs as ecs,
    aws_ecs_patterns as ecs_patterns,
    aws_codeartifact as codeartifact,
    aws_ssm as ssm,
    aws_ec2 as ec2,
    aws_elasticloadbalancingv2 as elbv2,
    aws_certificatemanager as acm,
    aws_iam as iam,
)
from constructs import Construct


class CodeArtifactProxy(Stack):
    """A CDK stack that creates the resources required for a Code Artifact Proxy (deployed as a load balanced fargate service)"""

    __ecs_service: ecs_patterns.ApplicationLoadBalancedFargateService = None
    vpc: ec2.Vpc = None

    domain_name: str = None
    repository_name: str = None
    domain_owner: str = None

    def __init__(
        self,
        scope: Construct,
        construct_id: str,
        domain_name: str,
        repository_name: str,
        domain_owner: str = None,
        vpc_id: str = None,
        **kwargs,
    ) -> None:
        super().__init__(scope, construct_id, **kwargs)

        self.domain_name = domain_name
        self.repository_name = repository_name
        self.domain_owner = domain_owner

        self.vpc = ec2.Vpc.from_lookup(
            self,
            "codeartifact_proxy_vpc",
            vpc_id=vpc_id,
        )

    def attach_iam_role(self):
        """Attaches an IAM role to the Fargate Container with some permissions to use codeartifact"""
        self.__ecs_service.task_definition.add_to_task_role_policy(
            statement=iam.PolicyStatement(
                actions=[
                    "codeartifact:Describe*",
                    "codeartifact:Get*",
                    "codeartifact:List*",
                    "codeartifact:ReadFromRepository",
                ],
                resources=[
                    cdk.Arn.format(
                        components=cdk.ArnComponents(
                            account=self.domain_owner,
                            service="logs",
                            resource="repository",
                            resource_name=f"{self.domain_name}/{self.repository_name}",
                        ),
                        stack=self,
                    )
                ],
            )
        )
        self.__ecs_service.task_definition.add_to_task_role_policy(
            statement=iam.PolicyStatement(
                actions=["sts:GetServiceBearerToken"],
                resources=["*"],
                conditions={
                    "StringEquals": {"sts:AWSServiceName": "codeartifact.amazonaws.com"}
                },
            )
        )

    def create_code_artifact(self):
        """Creates a CodeArtifact repository"""

        domain = codeartifact.CfnDomain(
            self,
            id="codeartifact_domain",
            domain_name=self.domain_name,
            encryption_key="alias/aws/codeartifact",
        )

        codeartifact.CfnRepository(
            self,
            id="codeartifact_repository",
            domain_name=domain.attr_name,
            repository_name=self.repository_name,
        )

    def create_loadbalanced_fargate(
        self,
        certificate_arn: str = None,
        certificate_ssm_parameter: str = None,
        subnet_group_name: str = "Applications",
    ):
        """Creates a Load Balanced fargate service with the Code Artifact proxy

        Args:
            certificate_arn (str, optional): The ARN of the ACM certificate to use for the load balanced service. Defaults to None.
            certificate_ssm_parameter (str, optional): The SSM parameter name that stores the ACM certificate ARN to use for the load balanced service. Defaults to None.
            subnet_group_name (str, optional): The name of the subnet group to use for the load balanced service. Defaults to "Applications".
        """
        cluster = ecs.Cluster(self, "codeartifact_ecs_cluster", vpc=self.vpc)

        task_image_options = ecs_patterns.ApplicationLoadBalancedTaskImageOptions(
            container_port=8080,
            image=ecs.ContainerImage.from_registry(
                "sktan/aws-codeartifact-proxy:latest"
            ),
        )

        certificate = None
        # Raise an error if both ACM certificate ARN and SSM parameter resolution are specified
        if certificate_arn and certificate_ssm_parameter:
            raise Exception(
                "Both certificate_arn and certificate_ssm_parameter cannot be set"
            )

        # Resolve the CDK certificate object via the certificate ARN or SSM parameter value
        if certificate_arn:
            certificate = acm.Certificate.from_certificate_arn(
                self, id="acm_certificate", certificate_arn=certificate_arn
            )
        elif certificate_ssm_parameter:
            certificate = acm.Certificate.from_certificate_arn(
                self,
                id="acm_certificate",
                certificate_arn=ssm.StringParameter.value_for_string_parameter(
                    self, parameter_name=certificate_ssm_parameter
                ),
            )
        protocol = (
            elbv2.ApplicationProtocol.HTTPS
            if certificate
            else elbv2.ApplicationProtocol.HTTP
        )

        self.__ecs_service = ecs_patterns.ApplicationLoadBalancedFargateService(
            self,
            "codeartifact_ecs_service",
            cluster=cluster,
            task_image_options=task_image_options,
            cpu=512,
            memory_limit_mib=1024,
            public_load_balancer=False,
            protocol=protocol,
            certificate=certificate,
            redirect_http=(protocol == elbv2.ApplicationProtocol.HTTPS),
            max_healthy_percent=100,
            min_healthy_percent=0,
        )

        if subnet_group_name:
            cfn_lb = self.__ecs_service.load_balancer.node.default_child
            cfn_lb.subnets = self.vpc.select_subnets(
                subnet_group_name=subnet_group_name
            ).subnet_ids

        self.attach_iam_role()
