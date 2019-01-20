// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
	spartaMage "github.com/mweagle/Sparta/magefile"
)

// BuildSite builds the ReactJS site
func BuildSite() error {
	curDir, curDirErr := os.Getwd()
	if curDirErr != nil {
		return curDirErr
	}
	// Run the react-scripts build command
	absPath, absPathErr := filepath.Abs("./aws-amplify-auth-starters")
	if absPathErr != nil {
		return absPathErr
	}
	chgDirErr := os.Chdir(absPath)
	if chgDirErr != nil {
		return chgDirErr
	}
	runErr := sh.Run("npm", "run-script", "build")
	restoreErr := os.Chdir(curDir)
	if restoreErr != nil {
		fmt.Fprintf(os.Stderr,
			"Failed to restore working directory: %s",
			restoreErr.Error())
	}
	return runErr
}

// Provision the service
func Provision() error {
	return spartaMage.Provision()
}

// Describe the stack by producing an HTML representation of the CloudFormation
// template
func Describe() error {
	return spartaMage.Describe()
}

// Delete the service, iff it exists
func Delete() error {
	return spartaMage.Delete()
}

// Status report if the stack has been provisioned
func Status() error {
	return spartaMage.Status()
}

// Version information
func Version() error {
	return spartaMage.Version()
}
