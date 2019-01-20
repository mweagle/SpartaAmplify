// +build lambdabinary

package bootstrap

import (
	"github.com/aws/aws-sdk-go/aws/session"
	sparta "github.com/mweagle/Sparta"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"
)

// NewServiceDecoratorHandler returns a new ServiceDecoratorHookHandler
// instance to annotate the template
func NewServiceDecoratorHandler(site *sparta.S3Site) (sparta.ServiceDecoratorHookHandler, error) {

	// Return a NOP decorator and no error
	nopDecorator := func(context map[string]interface{},
		serviceName string,
		template *gocf.Template,
		S3Bucket string,
		S3Key string,
		buildID string,
		awsSession *session.Session,
		noop bool,
		logger *logrus.Logger) error {
		return nil
	}
	return sparta.ServiceDecoratorHookFunc(nopDecorator), nil
}
