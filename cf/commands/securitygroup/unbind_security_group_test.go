package securitygroup_test

import (
	"errors"

	"github.com/cloudfoundry/cli/cf/api/apifakes"
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/models"

	"github.com/cloudfoundry/cli/cf/api/organizations/organizationsfakes"
	"github.com/cloudfoundry/cli/cf/api/securitygroups/securitygroupsfakes"
	"github.com/cloudfoundry/cli/cf/api/securitygroups/spaces/spacesfakes"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("unbind-security-group command", func() {
	var (
		ui                  *testterm.FakeUI
		securityGroupRepo   *securitygroupsfakes.FakeSecurityGroupRepo
		orgRepo             *organizationsfakes.FakeOrganizationRepository
		spaceRepo           *apifakes.FakeSpaceRepository
		secBinder           *spacesfakes.FakeSecurityGroupSpaceBinder
		requirementsFactory *testreq.FakeReqFactory
		configRepo          coreconfig.Repository
		deps                commandregistry.Dependency
	)

	updateCommandDependency := func(pluginCall bool) {
		deps.UI = ui
		deps.RepoLocator = deps.RepoLocator.SetSpaceRepository(spaceRepo)
		deps.RepoLocator = deps.RepoLocator.SetOrganizationRepository(orgRepo)
		deps.RepoLocator = deps.RepoLocator.SetSecurityGroupRepository(securityGroupRepo)
		deps.RepoLocator = deps.RepoLocator.SetSecurityGroupSpaceBinder(secBinder)
		deps.Config = configRepo
		commandregistry.Commands.SetCommand(commandregistry.Commands.FindCommand("unbind-security-group").SetDependency(deps, pluginCall))
	}

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		requirementsFactory = &testreq.FakeReqFactory{}
		securityGroupRepo = new(securitygroupsfakes.FakeSecurityGroupRepo)
		orgRepo = new(organizationsfakes.FakeOrganizationRepository)
		spaceRepo = new(apifakes.FakeSpaceRepository)
		secBinder = new(spacesfakes.FakeSecurityGroupSpaceBinder)
		configRepo = testconfig.NewRepositoryWithDefaults()
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCLICommand("unbind-security-group", args, requirementsFactory, updateCommandDependency, false)
	}

	Describe("requirements", func() {
		It("should fail if not logged in", func() {
			Expect(runCommand("my-group")).To(BeFalse())
		})

		It("should fail with usage when not provided with any arguments", func() {
			requirementsFactory.LoginSuccess = true
			runCommand()
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires", "arguments"},
			))
		})

		It("should fail with usage when provided with a number of arguments that is either 2 or 4 or a number larger than 4", func() {
			requirementsFactory.LoginSuccess = true
			runCommand("I", "like")
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires", "arguments"},
			))
			runCommand("Turn", "down", "for", "what")
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires", "arguments"},
			))
			runCommand("My", "Very", "Excellent", "Mother", "Just", "Sat", "Under", "Nine", "ThingsThatArentPlanets")
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires", "arguments"},
			))
		})
	})

	Context("when logged in", func() {
		BeforeEach(func() {
			requirementsFactory.LoginSuccess = true
		})

		Context("when everything exists", func() {
			BeforeEach(func() {
				securityGroup := models.SecurityGroup{
					SecurityGroupFields: models.SecurityGroupFields{
						Name:  "my-group",
						GUID:  "my-group-guid",
						Rules: []map[string]interface{}{},
					},
				}

				securityGroupRepo.ReadReturns(securityGroup, nil)

				orgRepo.ListOrgsReturns([]models.Organization{{
					OrganizationFields: models.OrganizationFields{
						Name: "my-org",
						GUID: "my-org-guid",
					}},
				}, nil)

				space := models.Space{SpaceFields: models.SpaceFields{Name: "my-space", GUID: "my-space-guid"}}
				spaceRepo.FindByNameInOrgReturns(space, nil)
			})

			It("removes the security group when we only pass the security group name (using the targeted org and space)", func() {
				runCommand("my-group")

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Unbinding security group", "my-org", "my-space", "my-user"},
					[]string{"OK"},
				))
				securityGroupGUID, spaceGUID := secBinder.UnbindSpaceArgsForCall(0)
				Expect(securityGroupGUID).To(Equal("my-group-guid"))
				Expect(spaceGUID).To(Equal("my-space-guid"))
			})

			It("removes the security group when we pass the org and space", func() {
				runCommand("my-group", "my-org", "my-space")

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Unbinding security group", "my-org", "my-space", "my-user"},
					[]string{"OK"},
				))
				securityGroupGUID, spaceGUID := secBinder.UnbindSpaceArgsForCall(0)
				Expect(securityGroupGUID).To(Equal("my-group-guid"))
				Expect(spaceGUID).To(Equal("my-space-guid"))
			})
		})

		Context("when one of the things does not exist", func() {
			BeforeEach(func() {
				securityGroupRepo.ReadReturns(models.SecurityGroup{}, errors.New("I accidentally the"))
			})

			It("fails with an error", func() {
				runCommand("my-group", "my-org", "my-space")
				Expect(ui.Outputs).To(ContainSubstrings([]string{"FAILED"}))
			})
		})
	})
})
