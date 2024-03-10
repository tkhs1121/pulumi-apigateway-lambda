package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Create an IAM role that AWS Lambda will use
		lambdaRole, err := iam.NewRole(ctx, "lambdaRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Action": "sts:AssumeRole",
					"Effect": "Allow",
					"Principal": {
						"Service": "lambda.amazonaws.com"
					}
				}]
			}`),
		})
		if err != nil {
			return err
		}

		// Attach the AWSLambdaBasicExecutionRole policy to the IAM role
		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaPolicyAttachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		// Define the AWS Lambda resource
		lambdaFunc, err := lambda.NewFunction(ctx, "helloLambda", &lambda.FunctionArgs{
			Code:    pulumi.NewFileArchive("function.zip"),
			Role:    lambdaRole.Arn,
			Handler: pulumi.String("bootstrap"),
			Runtime: pulumi.String("provided.al2"), 
		})
		if err != nil {
			return err
		}

		// Create an API Gateway Rest API
		api, err := apigateway.NewRestApi(ctx, "api", &apigateway.RestApiArgs{
			Description: pulumi.String("Example API"),
		})
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "apiGatewayInvoke", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  lambdaFunc.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
		})
		if err != nil {
			return err
		}

		// Create an API Gateway Resource
		resource, err := apigateway.NewResource(ctx, "resource", &apigateway.ResourceArgs{
			RestApi:  api.ID(),
			ParentId: api.RootResourceId,
			PathPart: pulumi.String("hello"),
		})
		if err != nil {
			return err
		}

		// Create an API Gateway Method for the 'GET' HTTP verb
		_, err = apigateway.NewMethod(ctx, "getMethod", &apigateway.MethodArgs{
			RestApi:       api.ID(),
			ResourceId:    resource.ID(),
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
		})
		if err != nil {
			return err
		}

		// Create an API Gateway Integration to connect the 'GET' method to the Lambda function
		integration, err := apigateway.NewIntegration(ctx, "getIntegration", &apigateway.IntegrationArgs{
			RestApi:               api.ID(),
			ResourceId:            resource.ID(),
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),      // Lambda functions are invoked with POST
			Type:                  pulumi.String("AWS_PROXY"), // Use the Lambda proxy integration
			Uri:                   lambdaFunc.InvokeArn,
		})
		if err != nil {
			return err
		}

		// Create a deployment to enable the API Gateway
		deployment, err := apigateway.NewDeployment(ctx, "deployment", &apigateway.DeploymentArgs{
			RestApi: api.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{
			integration,
		}))
		if err != nil {
			return err
		}

		// Create an API Gateway Stage which acts as an environment
		_, err = apigateway.NewStage(ctx, "stage", &apigateway.StageArgs{
			Deployment: deployment.ID(),
			RestApi:    api.ID(),
			StageName:  pulumi.String("prod"), // Name the stage as 'prod'
		})
		if err != nil {
			return err
		}

		// Output the invocation URL of the stage
		ctx.Export("invokeUrl", pulumi.Sprintf("https://%s.execute-api.%s.amazonaws.com/prod/hello", api.ID(), "ap-northeast-1"))

		return nil
	})
}
