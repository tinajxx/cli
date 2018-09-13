package experimental

import (
	"code.cloudfoundry.org/cli/integration/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/ghttp"
)

var _ = Describe("v3-zdt-push command", func() {
	var (
		orgName           string
		spaceName         string
		appName           string
		userName          string
		PublicDockerImage = "cloudfoundry/diego-docker-app-custom"
	)

	BeforeEach(func() {
		orgName = helpers.NewOrgName()
		spaceName = helpers.NewSpaceName()
		appName = helpers.PrefixedRandomName("app")
		userName, _ = helpers.GetCredentials()
		helpers.TurnOffExperimental()
	})

	AfterEach(func() {
		helpers.TurnOnExperimental()
	})

	Describe("help", func() {
		Context("when --help flag is set", func() {
			It("Displays command usage to output", func() {
				session := helpers.CF("v3-zdt-push", "--help")
				Eventually(session).Should(Say("NAME:"))
				Eventually(session).Should(Say("v3-zdt-push - Update an app with zero down time"))
				Eventually(session).Should(Say("USAGE:"))
				Eventually(session).Should(Say("cf v3-zdt-push APP_NAME \\[-b BUILDPACK\\]\\.\\.\\. \\[-p APP_PATH\\] \\[--no-route\\]"))
				Eventually(session).Should(Say("cf v3-zdt-push APP_NAME --docker-image \\[REGISTRY_HOST:PORT/\\]IMAGE\\[:TAG\\] \\[--docker-username USERNAME\\] \\[--no-route\\]"))
				Eventually(session).Should(Say("OPTIONS:"))
				Eventually(session).Should(Say("-b\\s+Custom buildpack by name \\(e\\.g\\. my-buildpack\\) or Git URL \\(e\\.g\\. 'https://github.com/cloudfoundry/java-buildpack.git'\\) or Git URL with a branch or tag \\(e\\.g\\. 'https://github.com/cloudfoundry/java-buildpack\\.git#v3.3.0' for 'v3.3.0' tag\\)\\. To use built-in buildpacks only, specify 'default' or 'null'"))
				Eventually(session).Should(Say("--docker-image, -o\\s+Docker image to use \\(e\\.g\\. user/docker-image-name\\)"))
				Eventually(session).Should(Say("--docker-username\\s+Repository username; used with password from environment variable CF_DOCKER_PASSWORD"))
				Eventually(session).Should(Say("--no-route\\s+Do not map a route to this app"))
				Eventually(session).Should(Say("-p\\s+Path to app directory or to a zip file of the contents of the app directory"))
				Eventually(session).Should(Say("ENVIRONMENT:"))
				Eventually(session).Should(Say("CF_DOCKER_PASSWORD=\\s+Password used for private docker repository"))
				Eventually(session).Should(Say("CF_STAGING_TIMEOUT=15\\s+Max wait time for buildpack staging, in minutes"))
				Eventually(session).Should(Say("CF_STARTUP_TIMEOUT=5\\s+Max wait time for app instance startup, in minutes"))

				Eventually(session).Should(Exit(0))
			})
		})
	})

	Context("when the app name is not provided", func() {
		It("tells the user that the app name is required, prints help text, and exits 1", func() {
			session := helpers.CF("v3-zdt-push")

			Eventually(session.Err).Should(Say("Incorrect Usage: the required argument `APP_NAME` was not provided"))
			Eventually(session).Should(Say("NAME:"))
			Eventually(session).Should(Exit(1))
		})
	})

	It("displays the experimental warning", func() {
		session := helpers.CF("v3-zdt-push", appName)
		Eventually(session.Err).Should(Say("This command is in EXPERIMENTAL stage and may change without notice"))
		Eventually(session).Should(Exit())
	})

	Context("when the environment is not setup correctly", func() {
		Context("when no API endpoint is set", func() {
			BeforeEach(func() {
				helpers.UnsetAPI()
			})

			It("fails with no API endpoint set message", func() {
				session := helpers.CF("v3-zdt-push", appName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No API endpoint set\\. Use 'cf login' or 'cf api' to target an endpoint\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when the v3 api does not exist", func() {
			var server *Server

			BeforeEach(func() {
				server = helpers.StartAndTargetServerWithoutV3API()
			})

			AfterEach(func() {
				server.Close()
			})

			It("fails with error message that the minimum version is not met", func() {
				session := helpers.CF("v3-zdt-push", appName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("This command requires CF API version 3\\.27\\.0 or higher\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when the v3 api version is lower than the minimum version", func() {
			var server *Server

			BeforeEach(func() {
				server = helpers.StartAndTargetServerWithAPIVersions(helpers.DefaultV2Version, "3.0.0")
			})

			AfterEach(func() {
				server.Close()
			})

			It("fails with error message that the minimum version is not met", func() {
				session := helpers.CF("v3-zdt-push", appName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("This command requires CF API version 3\\.27\\.0 or higher\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when not logged in", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
			})

			It("fails with not logged in message", func() {
				session := helpers.CF("v3-zdt-push", appName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("Not logged in\\. Use 'cf login' to log in\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when there is no org set", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
			})

			It("fails with no org targeted error message", func() {
				session := helpers.CF("v3-zdt-push", appName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No org targeted, use 'cf target -o ORG' to target an org\\."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when there is no space set", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
				helpers.TargetOrg(ReadOnlyOrg)
			})

			It("fails with no space targeted error message", func() {
				session := helpers.CF("v3-zdt-push", appName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No space targeted, use 'cf target -s SPACE' to target a space\\."))
				Eventually(session).Should(Exit(1))
			})
		})
	})

	Context("when the environment is set up correctly", func() {
		var domainName string

		BeforeEach(func() {
			helpers.SetupCF(orgName, spaceName)
			domainName = helpers.DefaultSharedDomain()
		})

		AfterEach(func() {
			helpers.QuickDeleteOrg(orgName)
		})

		Context("when the app exists", func() {
			var session *Session
			BeforeEach(func() {
				helpers.WithHelloWorldApp(func(appDir string) {
					Eventually(helpers.CustomCF(helpers.CFEnv{WorkingDirectory: appDir}, "v3-zdt-push", appName)).Should(Exit(0))
				})

				helpers.WithHelloWorldApp(func(appDir string) {
					session = helpers.CustomCF(helpers.CFEnv{WorkingDirectory: appDir}, "v3-zdt-push", appName, "-b", "https://github.com/cloudfoundry/staticfile-buildpack")
					Eventually(session).Should(Exit(0))
				})
			})

			It("pushes the app", func() {
				Eventually(session).Should(Say("Updating app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Uploading and creating bits package for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Staging package for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say("Setting app %s to droplet .+ in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Mapping routes\\.\\.\\."))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Starting app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Starting deployment for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Waiting for app to start\\.\\.\\."))
				Eventually(session).Should(Say("Showing health and status for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("name:\\s+%s", appName))
				Eventually(session).Should(Say("requested state:\\s+started"))
				Eventually(session).Should(Say("routes:\\s+%s\\.%s", appName, domainName))
				Eventually(session).Should(Say("stack:\\s+cflinuxfs2"))

				// TODO: Uncomment when capi sorts out droplet buildpack name/detectoutput
				// Eventually(session).Should(Say("buildpacks:\\s+https://github.com/cloudfoundry/staticfile-buildpack"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("type:\\s+web"))
				Eventually(session).Should(Say("instances:\\s+1/1"))
				Eventually(session).Should(Say("memory usage:\\s+32M"))
				Eventually(session).Should(Say(`state\s+since\s+cpu\s+memory\s+disk`))
				Eventually(session).Should(Say("#0\\s+running\\s+\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2} [AP]M"))
			})
		})

		Context("when the app does not already exist", func() {
			var session *Session

			BeforeEach(func() {
				helpers.WithHelloWorldApp(func(appDir string) {
					session = helpers.CustomCF(helpers.CFEnv{WorkingDirectory: appDir}, "v3-zdt-push", appName)
					Eventually(session).Should(Exit(0))
				})
			})

			It("pushes the app", func() {
				Eventually(session).Should(Say("Creating app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Uploading and creating bits package for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Consistently(session).ShouldNot(Say("Stopping app %s", appName))
				Eventually(session).Should(Say("Staging package for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say("Setting app %s to droplet .+ in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Mapping routes\\.\\.\\."))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Starting app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("Waiting for app to start\\.\\.\\."))
				Eventually(session).Should(Say("Showing health and status for app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("name:\\s+%s", appName))
				Eventually(session).Should(Say("requested state:\\s+started"))
				Eventually(session).Should(Say("routes:\\s+%s\\.%s", appName, domainName))
				Eventually(session).Should(Say("stack:\\s+cflinuxfs2"))
				Eventually(session).Should(Say("buildpacks:\\s+staticfile"))
				Eventually(session).Should(Say(""))
				Eventually(session).Should(Say("type:\\s+web"))
				Eventually(session).Should(Say("instances:\\s+1/1"))
				Eventually(session).Should(Say("memory usage:\\s+32M"))
				Eventually(session).Should(Say(`state\s+since\s+cpu\s+memory\s+disk`))
				Eventually(session).Should(Say("#0\\s+running\\s+\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2} [AP]M"))
			})
		})

	})
})
