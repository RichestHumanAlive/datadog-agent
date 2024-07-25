// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package installerwindows

import (
	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/e2e"
	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/environments/aws/host/windows"
	"github.com/DataDog/datadog-agent/test/new-e2e/tests/installer"
	"testing"
)

type testAgentInstallSuite struct {
	baseSuite
}

// TestAgentInstalls tests the usage of the Datadog Installer to install the Datadog Agent package.
func TestAgentInstalls(t *testing.T) {
	e2e.Run(t, &testAgentInstallSuite{},
		e2e.WithProvisioner(
			winawshost.ProvisionerNoAgentNoFakeIntake(
				winawshost.WithInstaller(),
			)),
		e2e.WithStackName("datadog-windows-installer-test"))
}

// TestInstallAgentPackage tests installing and uninstalling the Datadog Agent using the Datadog Installer.
func (suite *testAgentInstallSuite) TestInstallAgentPackage() {
	suite.Run("Install", func() {
		// Arrange

		// Act
		output, err := suite.installer.InstallPackage(AgentPackage)

		// Assert
		suite.Require().NoErrorf(err, "failed to install the Datadog Agent package: %s", output)
		suite.Require().Host(suite.Env().RemoteHost).HasARunningDatadogAgentService()
	})

	suite.Run("Uninstall", func() {
		// Arrange

		// Act
		output, err := suite.installer.RemovePackage(AgentPackage)

		// Assert
		suite.Require().NoErrorf(err, "failed to remove the Datadog Agent package: %s", output)
		suite.Require().Host(suite.Env().RemoteHost).HasNoDatadogAgentService()
	})
}

// TestUpgradeAgentPackage tests that it's possible to upgrade the Datadog Agent using the Datadog Installer.
func (suite *testAgentInstallSuite) TestUpgradeAgentPackage() {
	suite.Run("Install", func() {
		// Arrange

		// Act
		output, err := suite.installer.InstallPackage(AgentPackage, WithVersion(installer.StableVersionPackage))

		// Assert
		suite.Require().NoErrorf(err, "failed to install the Datadog Agent package: %s", output)
		suite.Require().Host(suite.Env().RemoteHost).
			HasBinary("C:\\Program Files\\Datadog\\Datadog Agent\\bin\\agent.exe").
			WithVersionEqual(installer.StableVersion).
			HasAService("datadogagent").
			WithStatus("Running")
	})

	suite.Run("Uninstall", func() {
		// Arrange

		// Act
		output, err := suite.installer.RemovePackage(AgentPackage)

		// Assert
		suite.Require().NoErrorf(err, "failed to remove the Datadog Agent package: %s", output)
		suite.Require().Host(suite.Env().RemoteHost).HasNoDatadogAgentService()
	})
}
