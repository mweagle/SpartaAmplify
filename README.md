# SpartaAmplify

**Note** : Requires Sparta v1.9.0 or higher

This is a Sparta application that leverages the AWS Amplify [UI Components](https://aws-amplify.github.io/docs/js/ui).

It builds on the [aws-amplify-auth-starters](https://github.com/aws-samples/aws-amplify-auth-starters/tree/react) project to
 an HTML GUI that uses Cognito-backed authorization.

Features:

- ReactJS `npm run build` is executed as a `go:generate` step via a [magefile](https://magefile.org/) task.
- Cognito resources are defined in a [ServiceDecoratorHook](https://godoc.org/github.com/mweagle/Sparta#ServiceDecoratorHook)
- Cognito resources use [go-cloudcondensor](https://github.com/mweagle/go-cloudcondenser) to streamline template definition. _go-cloudcondensor_ is similar to the [AWS CDK](https://github.com/awslabs/aws-cdk).
- The dynamic infrastructure values are included in the S3 Site _MANIFEST.JSON_, deployed to the S3 bucket, and fetched by [whatwg-fetch](https://www.npmjs.com/package/whatwg-fetch).


## Usage

To build this project:

1. `git clone https://github.com/mweagle/SpartaAmplify`
1. `cd SpartaAmplify/aws-amplify-auth-starters`
1. `npm install`
1. `cd ..`
1. `go run main.go provision --s3Bucket $MY_BUCKET`

Visit the **S3SiteURL** URL Output to view your React app.

**NOTE**: Phone number signup requires a [specific format](https://docs.aws.amazon.com/cognito/latest/developerguide/user-pool-settings-attributes.html).

Example: `+12065551212`

See the [React starter app](https://github.com/aws-samples/aws-amplify-auth-starters/tree/react) for more information.

## Results

<div align="center"><img src="https://raw.githubusercontent.com/mweagle/SpartaAmplify/master/readme/sparta_react_app.jpg" />
</div>

<div align="center"><img src="https://raw.githubusercontent.com/mweagle/SpartaAmplify/master/readme/sparta_react_app_signup.jpg" />
</div>

## Infrastructure

<div align="center"><img src="https://raw.githubusercontent.com/mweagle/SpartaAmplify/master/readme/template-designer.png" />
</div>
