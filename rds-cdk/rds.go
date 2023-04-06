package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

var dbClusterName = "AcornRdsCluster"

func getItemName(item string) *string {
	return jsii.String(fmt.Sprintf("%s%s", dbClusterName, item))
}

func NewRDSStack(scope constructs.Construct, id string, props *awscdk.StackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = *props
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := awsec2.Vpc_FromLookup(stack, jsii.String("VPC"), &awsec2.VpcLookupOptions{
		VpcId: jsii.String("vpc-0a95e30e5c79ce188"),
	})

	sg := awsec2.NewSecurityGroup(stack, getItemName("SG"), &awsec2.SecurityGroupProps{
		Vpc:              vpc,
		AllowAllOutbound: jsii.Bool(true),
		Description:      jsii.String("Acorn created Rds security group"),
	})

	subnetGroup := awsrds.NewSubnetGroup(stack, getItemName("SubnetGroup"), &awsrds.SubnetGroupProps{
		Description: jsii.String("RDS SUBNETS..."),
		Vpc:         vpc,
		VpcSubnets: &awsec2.SubnetSelection{
			SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
		},
	})

	for _, i := range *vpc.PrivateSubnets() {
		sg.AddIngressRule(awsec2.Peer_Ipv4(i.Ipv4CidrBlock()), awsec2.Port_Tcp(jsii.Number(3306)), jsii.String("Allow from private subnets"), jsii.Bool(false))
	}
	for _, i := range *vpc.PublicSubnets() {
		sg.AddIngressRule(awsec2.Peer_Ipv4(i.Ipv4CidrBlock()), awsec2.Port_Tcp(jsii.Number(3306)), jsii.String("Allow from public subnets"), jsii.Bool(false))
	}
	sgs := &[]awsec2.ISecurityGroup{sg}

	creds := awsrds.Credentials_FromGeneratedSecret(jsii.String("clusteradmin"), &awsrds.CredentialsBaseOptions{})

	cluster := awsrds.NewServerlessCluster(stack, getItemName(id), &awsrds.ServerlessClusterProps{
		Engine: awsrds.DatabaseClusterEngine_AURORA_MYSQL(),

		CopyTagsToSnapshot: jsii.Bool(true),
		Credentials:        creds,
		Vpc:                vpc,
		Scaling: &awsrds.ServerlessScalingOptions{
			AutoPause: awscdk.Duration_Minutes(jsii.Number(10)),
		},
		SubnetGroup:    subnetGroup,
		SecurityGroups: sgs,
	})

	awscdk.Tags_Of(cluster).Add(jsii.String("AcornSVC"), getItemName("-Owned"), &awscdk.TagProps{})

	port := "3306"
	pSlice := strings.SplitN(*cluster.ClusterEndpoint().SocketAddress(), ":", 2)
	if len(pSlice) == 2 {
		port = pSlice[1]
	}

	awscdk.NewCfnOutput(stack, getItemName("-host"), &awscdk.CfnOutputProps{
		Value: cluster.ClusterEndpoint().Hostname(),
	})
	awscdk.NewCfnOutput(stack, getItemName("-port"), &awscdk.CfnOutputProps{
		Value: &port,
	})
	awscdk.NewCfnOutput(stack, getItemName("-username"), &awscdk.CfnOutputProps{
		Value: creds.Username(),
	})
	awscdk.NewCfnOutput(stack, getItemName("-password-arn"), &awscdk.CfnOutputProps{
		Value: cluster.Secret().SecretArn(),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	NewRDSStack(app, "bill", &awscdk.StackProps{
		Env: rdsenv(),
	})

	app.Synth(nil)
}

func rdsenv() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
