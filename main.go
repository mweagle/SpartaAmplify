package main

//go:generate mage -v BuildSite

import (
	"context"
	"fmt"

	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	"github.com/mweagle/SpartaAmplify/bootstrap"
	"github.com/sirupsen/logrus"
)

/*
================================================================================
╦  ╔═╗╔╦╗╔╗ ╔╦╗╔═╗
║  ╠═╣║║║╠╩╗ ║║╠═╣
╩═╝╩ ╩╩ ╩╚═╝═╩╝╩ ╩
================================================================================
*/
func helloWorld(ctx context.Context) (interface{}, error) {
	logger, loggerOk := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)
	if loggerOk {
		logger.Info("Accessing structured logger 🙌")
	}
	contextLogger, contextLoggerOk := ctx.Value(sparta.ContextKeyRequestLogger).(*logrus.Entry)
	if contextLoggerOk {
		contextLogger.Info("Accessing request-scoped log, with request ID field")
	} else if loggerOk {
		logger.Warn("Failed to access scoped logger")
	} else {
		fmt.Printf("Failed to access any logger")
	}
	return "Hello World 👋. Welcome to AWS Lambda! 🙌🎉🍾", nil
}

/*
================================================================================
╔═╗╔═╗╔═╗╦  ╦╔═╗╔═╗╔╦╗╦╔═╗╔╗╔
╠═╣╠═╝╠═╝║  ║║  ╠═╣ ║ ║║ ║║║║
╩ ╩╩  ╩  ╩═╝╩╚═╝╩ ╩ ╩ ╩╚═╝╝╚╝
================================================================================
*/

func main() {

	lambdaFn, lambdaFnErr := sparta.NewAWSLambda("Hello World",
		helloWorld,
		sparta.IAMRoleDefinition{})
	if lambdaFnErr != nil {
		panic("Failed to create lambda function: " + lambdaFnErr.Error())
	}
	// Provision an S3 site
	s3Site, s3SiteErr := sparta.NewS3Site("./aws-amplify-auth-starters/build")
	if s3SiteErr != nil {
		panic("Failed to create S3 Site")
	}

	// Annotate this stack with all the Roles, Cognito pools necessary
	// to handle the authentication...
	decoratorFunc, decoratorFuncErr := bootstrap.NewServiceDecoratorHandler(s3Site)
	if decoratorFuncErr != nil {
		panic("Failed to create bootstrapper")
	}

	// Sanitize the name so that it doesn't have any spaces
	lambdaFunctions := []*sparta.LambdaAWSInfo{
		lambdaFn,
	}
	workflowHooks := sparta.WorkflowHooks{
		ServiceDecorators: []sparta.ServiceDecoratorHookHandler{decoratorFunc},
	}
	// Define the stack
	stackName := spartaCF.UserScopedStackName("SpartaAmplify")
	sparta.MainEx(stackName,
		fmt.Sprintf("ReactJS Amplify app with authentication support"),
		lambdaFunctions,
		nil,
		s3Site,
		&workflowHooks,
		false)
}
