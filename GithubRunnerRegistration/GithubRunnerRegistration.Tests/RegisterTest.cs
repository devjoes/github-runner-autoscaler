using GithubRunnerRegistration.Models;
using Moq;
using Octokit;
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace GithubRunnerRegistration.Tests
{
    public class RegisterTest
    {
        const string Pat = "pat";
        const string Owner = "owner";
        const string Repo = "repo";


        [Fact]
        public async Task ShouldErrorIfRequestLabelsContainsInvalidChars()
        {
            Assert.Throws<InvalidOperationException>(() => new Register(new RegistrationRequest { AdminPat = Pat, Labels = new string[] { "foo\"" }, Owner = Owner, Repository = Repo }, "a"));
        }


        [Fact]
        public async Task ShouldErrorIfNameContainsInvalidChars()
        {
            var register = new Register(new RegistrationRequest { AdminPat = Pat, Labels = new string[] { }, Owner = Owner, Repository = Repo }, "a");
            await Assert.ThrowsAsync<InvalidOperationException>(async () => await register.AddRunner("foo\""));
        }

        [Fact]
        public async Task ShouldErrorIfRepoDoesNotExist()
        {
            var request = new RegistrationRequest { AdminPat = Pat, Labels = new string[] { }, Owner = Owner, Repository = Repo };
            var register = new Register(request, Assembly.GetExecutingAssembly().Location);
            var mockClient = new Mock<IGitHubClient>();
            mockClient.Setup(c => c.Repository.Get(Owner, Repo)).Throws(new Exception("boom"));
            
            await Assert.ThrowsAsync<SetupException>(async()=> await register.Setup(mockClient.Object));
        }

        [Fact]
        public async Task ShouldErrorIfTokenIsNotAdmin()
        {
            var request = new RegistrationRequest { AdminPat = Pat, Labels = new string[] { }, Owner = Owner, Repository = Repo };
            var register = new Register(request, Assembly.GetExecutingAssembly().Location);
            var mockClient = new Mock<IGitHubClient>();
            var perms = new RepositoryPermissions(false, true, true);
            mockClient.Setup(c => c.Repository.Get(Owner, Repo)).ReturnsAsync(new Repository(default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, perms, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default));
            await Assert.ThrowsAsync<SetupException>(async () => await register.Setup(mockClient.Object));
        }

        [Fact]
        public async Task ShouldSetRunnerRegistrationToken()
        {
            const string token = "foo";
            var request = new RegistrationRequest { AdminPat = Pat, Labels = new string[] { "foo"}, Owner = Owner, Repository = Repo };
            var register = new Register(request, Assembly.GetExecutingAssembly().Location);
            var mockClient = new Mock<IGitHubClient>();
            var perms = new RepositoryPermissions(true, true, true);
            mockClient.Setup(c => c.Repository.Get(Owner, Repo)).ReturnsAsync(new Repository(default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, perms, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default, default));
            var response = new Mock<IApiResponse<TokenResult>>();
            response.SetupGet(r => r.Body).Returns(new TokenResult { Token = token });
            mockClient.Setup(c => c.Connection.Post<TokenResult>(new Uri($"https://api.github.com/repos/{Owner}/{Repo}/actions/runners/registration-token"), It.IsAny<CancellationToken>())).Returns(Task.FromResult(response.Object)).Verifiable();
            await register.Setup(mockClient.Object);
            Assert.Equal(token, register.RunnerRegistrationToken);
            mockClient.Verify();
        }

        [Fact]
        public async Task ShouldCopyAndExecuteBinary()
        {
            const string binaryName = "binary";
            const string name = "foobar";
            var tmp = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
            Directory.CreateDirectory(tmp);
            var binary = Path.Combine(tmp, binaryName);
            File.WriteAllText(binary, "this isn't actually executable");

            var request = new RegistrationRequest { AdminPat = Pat, Labels = new string[] {"foo" }, Owner = Owner, Repository = Repo };
            var register = new Register(request, binary);
            register.RunnerRegistrationToken = "RunnerRegistrationToken";
            var secretGenerator = new Mock<IGetCredentials>();
            string credsDir = Path.Combine(Path.GetTempPath(), name);
            string runnerWd = Path.Combine(credsDir, "runner");            
            secretGenerator.Setup(g => g.GetCredentialsFromPath(credsDir)).Returns(Task.FromResult(new Dictionary<string, string>{ { "foo", "bar" } })).Verifiable();
            void mockRun(ProcessStartInfo startInfo)
            {
                Assert.Equal("dotnet", startInfo.FileName);
                Assert.Equal(runnerWd, startInfo.WorkingDirectory);
                Assert.True(Directory.Exists(runnerWd));
                Assert.True(File.Exists(Path.Combine(runnerWd, binaryName)));
                Assert.Equal($"{binaryName} configure --name \"{name}\" --token {register.RunnerRegistrationToken} --url \"https://github.com/{Owner}/{Repo}\" --labels \"foo\" --replace --unattended", startInfo.Arguments);
            }
            var creds = await register.RegisterRunner(name, secretGenerator.Object, mockRun);
            Assert.Equal("bar", creds["foo"]);
            Assert.False(Directory.Exists(runnerWd));
            secretGenerator.Verify();
            Directory.Delete(tmp, true);
        }

    }
}
