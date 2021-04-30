using GithubRunnerRegistration.Models;
using Octokit;
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading.Tasks;

namespace GithubRunnerRegistration
{
    public class Register
    {
        private readonly RegistrationRequest request;
        private readonly FileInfo binary;

        public string RunnerRegistrationToken { get; set; }

        public Register(RegistrationRequest request, string binary = "/actions-runner/bin/Runner.Listener.dll")
        {
            Array.ForEach(request.Labels, this.validateString);
            this.request = request;
            this.binary = new FileInfo(binary);
        }

        private GitHubClient getClient() => new GitHubClient(new ProductHeaderValue(nameof(GithubRunnerRegistration))) { Credentials = new Credentials(this.request.AdminPat) };

        private void validateString(string str)
        {
            var rx = "[^0-9a-z-_]";
            if (Regex.IsMatch(str, rx, RegexOptions.IgnoreCase))
            {
                throw new InvalidOperationException($"'{str}' contains invalid chars (rx: {rx})");
            }
        }

        public async Task Setup(IGitHubClient clnt = default)
        {
            var client = clnt != default ? clnt : this.getClient();
            Repository repository;
            try
            {
                repository = await client.Repository.Get(this.request.Owner, this.request.Repository);
            }
            catch (Exception ex)
            {
                throw new SetupException("Error getting repository", ex);
            }
            if (!repository.Permissions.Admin)
            {
                throw new SetupException("Not admin", default);
            }
            var tokenResult = await client.Connection.Post<TokenResult>(new Uri($"https://api.github.com/repos/{this.request.Owner}/{this.request.Repository}/actions/runners/registration-token"));
            this.RunnerRegistrationToken = tokenResult.Body.Token;
        }

        public async Task<RunnerRegistrationSecretData> AddRunner(string name, bool dryRun) => await this.RegisterRunner(name, new GetCredentials(), dryRun, this.Run);

        private void Run(ProcessStartInfo arg)
        {
            var proc = Process.Start(arg);
            proc.WaitForExit();
            if (proc.ExitCode != 0)
            {
                throw new Exception("Binary returned code " + proc.ExitCode);
            }
        }

        public async Task<RunnerRegistrationSecretData> RegisterRunner(string name, IGetCredentials getCreds, bool dryRun, Action<ProcessStartInfo> run)
        {
            this.validateString(name);
            var tmp = Path.Combine(Path.GetTempPath(), name);
            if (Directory.Exists(tmp))
            {
                Directory.Delete(tmp, true);
            }
            Directory.CreateDirectory(tmp);
            DirectoryCopy(binary.Directory.FullName, Path.Combine(tmp, "runner"));

            var startInfo = new ProcessStartInfo("dotnet", $"{this.binary.Name} configure --name \"{name}\" --token {this.RunnerRegistrationToken} --url \"https://github.com/{this.request.Owner}/{this.request.Repository}\" --labels \"{string.Join(",", this.request.Labels)}\" --replace --unattended");
            startInfo.WorkingDirectory = Path.Combine(tmp, "runner");
            try
            {
                if (dryRun)
                {
                    return this.createDryRunRunner();
                }
                run(startInfo);
                var creds = await getCreds.GetCredentialsFromPath(tmp);
                
                return creds;
            }
            finally
            {
                Directory.Delete(tmp, true);
            }
        }

        private RunnerRegistrationSecretData createDryRunRunner()
        {
            var data = Convert.ToBase64String(Encoding.Default.GetBytes("{\"dryRun\":1}"));
            return new RunnerRegistrationSecretData
            {
                Id = Guid.NewGuid(),
                Credentials = data,
                CredentialsRsaParams = data,
                PrivatePem = data,
                PublicPem = data
            };
        }

        private static void DirectoryCopy(string sourceDirName, string destDirName)
        {
            DirectoryInfo dir = new DirectoryInfo(sourceDirName);

            if (!dir.Exists)
            {
                throw new DirectoryNotFoundException(
                    "Source directory does not exist or could not be found: "
                    + sourceDirName);
            }

            DirectoryInfo[] dirs = dir.GetDirectories();
            Directory.CreateDirectory(destDirName);
            FileInfo[] files = dir.GetFiles();
            foreach (FileInfo file in files)
            {
                string tempPath = Path.Combine(destDirName, file.Name);
                file.CopyTo(tempPath, false);
            }
            foreach (DirectoryInfo subdir in dirs)
            {
                string tempPath = Path.Combine(destDirName, subdir.Name);
                DirectoryCopy(subdir.FullName, tempPath);
            }
        }
    }
}
