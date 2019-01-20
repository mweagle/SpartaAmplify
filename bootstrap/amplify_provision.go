// +build !lambdabinary

package bootstrap

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	sparta "github.com/mweagle/Sparta"
	spartaIAM "github.com/mweagle/Sparta/aws/iam"
	iamBuilder "github.com/mweagle/Sparta/aws/iam/builder"
	gocc "github.com/mweagle/go-cloudcondenser"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var calloutScript = `const response = require('cfn-response')
const aws = require('aws-sdk')
const identity = new aws.CognitoIdentityServiceProvider()
exports.handler = (event, context, callback) => {
	if (event.RequestType == 'Delete') {
		response.send(event, context, response.SUCCESS, {})
	}
	if (event.RequestType == 'Update' || event.RequestType == 'Create') {
		console.log("PARAMS: " + JSON.stringify(event, null, ' '));
		const params = {
			ClientId: event.ResourceProperties.ClientID,
			UserPoolId: event.ResourceProperties.UserpoolID
		}
		identity.describeUserPoolClient(params).promise()
			.then((res) => {
				response.send(event,
					context,
					response.SUCCESS,
					{'appSecret': res.UserPoolClient.ClientSecret})
			})
			.catch((err) => {
				response.send(event, context, response.FAILURE, {err})
			})
	}
}`

type lambdaCallout struct {
	gocf.CloudFormationCustomResource
	ClientID   gocf.Stringable
	UserpoolID gocf.Stringable
}

func (lc *lambdaCallout) CfnResourceType() string {
	return "Custom::LambdaCallout"
}

func (lc *lambdaCallout) CfnResourceAttributes() []string {
	return nil
}

func cognitoRole(roleType string) gocf.ResourceProperties {

	cognitoRole := &gocf.IAMRole{}
	cognitoRole.AssumeRolePolicyDocument = map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []spartaIAM.PolicyStatement{
			iamBuilder.Allow("sts:AssumeRoleWithWebIdentity").
				WithCondition(map[string]interface{}{
					"ForAnyValue:StringLike": map[string]string{
						"cognito-identity.amazonaws.com:amr": roleType,
					},
				}).
				ForFederatedPrincipals("cognito-identity.amazonaws.com").
				ToPolicyStatement(),
		},
	}
	return cognitoRole
}

func snsPublishRole(externalRoleID string) gocf.ResourceProperties {
	snsPublishRole := &gocf.IAMRole{}

	snsPublishRole.AssumeRolePolicyDocument = map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []spartaIAM.PolicyStatement{
			iamBuilder.Allow("sts:AssumeRole").
				WithCondition(map[string]interface{}{
					"StringEquals": map[string]string{
						"sts:ExternalId": externalRoleID,
					},
				}).
				ForPrincipals("cognito-idp.amazonaws.com").
				ToPolicyStatement(),
		},
	}
	snsPublishRole.Policies = &gocf.IAMRolePolicyList{
		gocf.IAMRolePolicy{
			PolicyName: gocf.String("cognitoauthservice_sns-role-dev"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": iamBuilder.
					Allow("sns:Publish").
					ForResource().
					Literal("*").
					ToPolicyStatement(),
			},
		},
	}
	return snsPublishRole
}

func userPool(snsResourceName string, externalID string) gocf.ResourceProperties {
	userPool := &gocf.CognitoUserPool{
		Schema: &gocf.CognitoUserPoolSchemaAttributeList{
			gocf.CognitoUserPoolSchemaAttribute{
				Name:     gocf.String("email"),
				Required: gocf.Bool(true),
				Mutable:  gocf.Bool(true),
			},
		},
		AutoVerifiedAttributes: gocf.StringList(
			gocf.String("email"),
		),
		EmailVerificationMessage: gocf.String("Your verification code is {####}"),
		EmailVerificationSubject: gocf.String("Your verification code"),
		Policies: &gocf.CognitoUserPoolPolicies{
			PasswordPolicy: &gocf.CognitoUserPoolPasswordPolicy{
				MinimumLength:    gocf.Integer(8),
				RequireLowercase: gocf.Bool(true),
				RequireNumbers:   gocf.Bool(true),
				RequireSymbols:   gocf.Bool(true),
				RequireUppercase: gocf.Bool(true),
			},
		},
		MfaConfiguration:       gocf.String("OFF"),
		SmsVerificationMessage: gocf.String("Your verification code is {####}"),
		SmsConfiguration: &gocf.CognitoUserPoolSmsConfiguration{
			SnsCallerArn: gocf.GetAtt(snsResourceName, "Arn"),
			ExternalID:   gocf.String(externalID),
		},
	}
	return userPool
}

func userPoolClientWeb(userPoolResourceName string) gocf.ResourceProperties {
	return &gocf.CognitoUserPoolClient{
		ClientName:           gocf.String("cognitoauthservice_app_clientWeb"),
		RefreshTokenValidity: gocf.Integer(30),
		UserPoolID:           gocf.Ref(userPoolResourceName).String(),
	}
}

func userPoolClient(userPoolResourceName string) gocf.ResourceProperties {
	userPoolClient := &gocf.CognitoUserPoolClient{
		ClientName:           gocf.String("cognitoauthservice_app_client"),
		GenerateSecret:       gocf.Bool(true),
		RefreshTokenValidity: gocf.Integer(30),
		UserPoolID:           gocf.Ref(userPoolResourceName).String(),
	}
	return userPoolClient
}

func userPoolClientRole() gocf.ResourceProperties {
	userPoolClientRole := &gocf.IAMRole{}

	userPoolClientRole.AssumeRolePolicyDocument = map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": iamBuilder.Allow("sts:AssumeRole").
			ForPrincipals("lambda.amazonaws.com").
			ToPolicyStatement(),
	}
	userPoolClientRole.Policies = &gocf.IAMRolePolicyList{
		gocf.IAMRolePolicy{
			PolicyDocument: sparta.ArbitraryJSONObject{
				"Version":   "2012-10-17",
				"Statement": sparta.CommonIAMStatements.Core,
			},
			PolicyName: gocf.String("LambdaCalloutPolicy"),
		},
	}
	return userPoolClientRole
}
func userPoolClientLambda(userPoolClientRole string) gocf.ResourceProperties {
	lambdaFunc := gocf.LambdaFunction{
		Handler: gocf.String("index.handler"),
		Runtime: gocf.String("nodejs6.10"),
		Timeout: gocf.Integer(300),
		Role:    gocf.GetAtt(userPoolClientRole, "Arn"),
		Code: &gocf.LambdaFunctionCode{
			ZipFile: gocf.String(calloutScript),
		},
	}
	return lambdaFunc
}

func userPoolClientLambdaPolicy(userRoleName string, userPoolName string) gocf.ResourceProperties {

	userPoolClientRole := &gocf.IAMPolicy{
		PolicyName: gocf.String("cognitoauthservice_userpoolclient_lambda_iam_policy"),
		Roles: gocf.StringList(
			gocf.Ref(userRoleName).String(),
		),
		PolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": iamBuilder.Allow("cognito-idp:DescribeUserPoolClient").
				ForResource().Attr(userPoolName, "Arn").
				ToPolicyStatement(),
		},
	}
	return userPoolClientRole
}

func userPoolClientLogPolicy(userRoleName string,
	userPoolClientLambda string) gocf.ResourceProperties {

	userPoolClientRole := &gocf.IAMPolicy{
		PolicyName: gocf.String("cognitoauthservice_userpoolclient_lambda_log_policy"),
		Roles: gocf.StringList(
			gocf.Ref(userRoleName).String(),
		),
		PolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": iamBuilder.Allow("logs:CreateLogGroup",
				"logs:CreateLogStream",
				"logs:PutLogEvents").ForResource().
				Literal("arn:aws:logs:").
				Region(":").
				AccountID(":").
				Literal("log-group:/aws/lambda/").
				Literal(userPoolClientLambda).
				Literal(":log-stream:*").
				ToPolicyStatement(),
		},
	}
	return userPoolClientRole
}

func userPoolClientInputs(calloutLambdaFunc string,
	userPoolClientResource string,
	userPoolResource string) gocf.ResourceProperties {

	// So for this I can just create a compatible object to stuff into
	// the template...
	calloutResource := &lambdaCallout{
		CloudFormationCustomResource: gocf.CloudFormationCustomResource{
			ServiceToken: gocf.GetAtt(calloutLambdaFunc, "Arn"),
		},
		ClientID:   gocf.Ref(userPoolClientResource).String(),
		UserpoolID: gocf.Ref(userPoolResource).String(),
	}
	return calloutResource
}

func identityPool(userPoolResource string,
	userPoolClientResource string,
	userPoolClientWebResource string) gocf.ResourceProperties {
	identityPool := &gocf.CognitoIdentityPool{
		CognitoIdentityProviders: &gocf.CognitoIdentityPoolCognitoIdentityProviderList{
			gocf.CognitoIdentityPoolCognitoIdentityProvider{
				ClientID: gocf.Ref(userPoolClientResource).String(),
				ProviderName: gocf.Join("",
					gocf.String("cognito-idp."),
					gocf.Ref("AWS::Region"),
					gocf.String(".amazonaws.com/"),
					gocf.Ref(userPoolResource)),
			},
			gocf.CognitoIdentityPoolCognitoIdentityProvider{
				ClientID: gocf.Ref(userPoolClientWebResource).String(),
				ProviderName: gocf.Join("",
					gocf.String("cognito-idp."),
					gocf.Ref("AWS::Region"),
					gocf.String(".amazonaws.com/"),
					gocf.Ref(userPoolResource)),
			},
		},
		AllowUnauthenticatedIDentities: gocf.Bool(false),
	}
	return identityPool
}
func identityPoolRoleMap(identityPoolResource string,
	authRoleResource string,
	unauthRoleResource string) gocf.ResourceProperties {
	roleAttach := &gocf.CognitoIdentityPoolRoleAttachment{
		IdentityPoolID: gocf.Ref(identityPoolResource).String(),
		Roles: map[string]interface{}{
			"unauthenticated": gocf.GetAtt(unauthRoleResource, "Arn"),
			"authenticated":   gocf.GetAtt(authRoleResource, "Arn"),
		},
	}
	return roleAttach
}

// NewServiceDecoratorHandler returns a new ServiceDecoratorHookHandler
// instance to annotate the template
func NewServiceDecoratorHandler(site *sparta.S3Site) (sparta.ServiceDecoratorHookHandler, error) {

	// Return a NOP decorator and no error
	amplifyDecorator := func(ctx map[string]interface{},
		serviceName string,
		template *gocf.Template,
		S3Bucket string,
		S3Key string,
		buildID string,
		awsSession *session.Session,
		noop bool,
		logger *logrus.Logger) error {

		// Let's go ahead and create some resources. We'll pretend this is a full
		// fledged stack and then just extract the resources...
		amplifyStackTemplate := gocc.CloudFormationCondenser{
			Description: "Amplify resources",
			Resources: []interface{}{
				// The Root roles
				gocc.Static("AuthRole",
					cognitoRole("authenticated")),
				gocc.Static("UnauthRole",
					cognitoRole("unauthenticated")),
				// The Cognito pools...
				gocc.Static("SNSRole",
					snsPublishRole("cognitoauthservice_role_external_id")),
				gocc.Static("UserPool",
					userPool("SNSRole", "cognitoauthservice_role_external_id")),
				gocc.Static("UserPoolClientWeb",
					userPoolClientWeb("UserPool")),
				gocc.Static("UserPoolClient",
					userPoolClient("UserPool")),
				gocc.Static("UserPoolClientRole",
					userPoolClientRole()),
				gocc.Static("UserPoolClientLambda",
					userPoolClientLambda("UserPoolClientRole")),
				gocc.Static("UserPoolClientLambdaPolicy",
					userPoolClientLambdaPolicy("UserPoolClientRole",
						"UserPool")),
				gocc.Static("UserPoolClientLogPolicy",
					userPoolClientLogPolicy("UserPoolClientRole",
						"UserPoolClientLambda")),
				gocc.Static("UserPoolClientInputs",
					userPoolClientInputs("UserPoolClientLambda",
						"UserPoolClient",
						"UserPool")),
				gocc.Static("IdentityPool", identityPool("UserPool",
					"UserPoolClient",
					"UserPoolClientWeb")),
				gocc.Static("IdentityPoolRoleMap",
					identityPoolRoleMap("IdentityPool",
						"AuthRole",
						"UnauthRole")),
			},
		}

		// Evaluate it...
		evalContext := context.Background()
		outputTemplate, outputErr := amplifyStackTemplate.Evaluate(evalContext)
		if outputErr != nil {
			return outputErr
		}
		mergeErrors := gocc.SafeMerge(outputTemplate, template)
		if len(mergeErrors) != 0 {
			return errors.Errorf("Failed to merge resources: %#v", mergeErrors)
		}
		// Utility function to associate dependencies with resources
		dependsOn := func(srcResource string, depResources ...string) {
			resourceProp, resourcePropOk := outputTemplate.Resources[srcResource]
			if resourcePropOk {
				resourceProp.DependsOn = depResources
			} else {
				logger.WithField("Resource", srcResource).Error("Resource not found")
			}
		}
		// Dependencies
		dependsOn("UserPoolClientWeb", "UserPool")
		dependsOn("UserPoolClient", "UserPool")
		dependsOn("UserPoolClientRole", "UserPoolClient")
		dependsOn("UserPoolClientLambda", "UserPoolClientRole")
		dependsOn("UserPoolClientLambdaPolicy", "UserPoolClientLambda")
		dependsOn("UserPoolClientLogPolicy", "UserPoolClientLambdaPolicy")
		dependsOn("UserPoolClientInputs", "UserPoolClientLogPolicy")
		dependsOn("IdentityPool", "UserPoolClientInputs")
		dependsOn("IdentityPoolRoleMap", "IdentityPool")

		// If everything is good then add some data to the MANIFEST
		authData := map[string]interface{}{
			"identityPoolId":      gocf.Ref("IdentityPool"),
			"region":              gocf.Ref("AWS::Region"),
			"userPoolId":          gocf.Ref("UserPool"),
			"userPoolWebClientId": gocf.Ref("UserPoolClientWeb"),
		}
		site.UserManifestData["Auth"] = authData

		// And finally add some outputs
		template.Outputs["IdentityPoolId"] = &gocf.Output{
			Value:       gocf.Ref("IdentityPool"),
			Description: "Id for the identity pool",
		}
		template.Outputs["IdentityPoolName"] = &gocf.Output{
			Value:       gocf.GetAtt("IdentityPool", "Name"),
			Description: "Name of the identity pool",
		}
		template.Outputs["UserPoolId"] = &gocf.Output{
			Value:       gocf.Ref("UserPool"),
			Description: "Id for the user pool",
		}
		template.Outputs["UserPoolArn"] = &gocf.Output{
			Value:       gocf.GetAtt("UserPool", "Arn"),
			Description: "ARN of the user pool",
		}
		template.Outputs["AppClientIDWeb"] = &gocf.Output{
			Value:       gocf.Ref("UserPoolClientWeb"),
			Description: "The user pool app client id for web",
		}
		template.Outputs["AppClientID"] = &gocf.Output{
			Value:       gocf.Ref("UserPoolClient"),
			Description: "The user pool app client id",
		}
		template.Outputs["AppClientSecret"] = &gocf.Output{
			Value:       gocf.GetAtt("UserPoolClientInputs", "appSecret"),
			Description: "The user pool app client id",
		}
		return nil
	}
	return sparta.ServiceDecoratorHookFunc(amplifyDecorator), nil
}
