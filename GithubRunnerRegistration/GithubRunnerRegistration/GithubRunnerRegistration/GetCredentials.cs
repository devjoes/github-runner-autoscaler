using GithubRunnerRegistration.Models;
using System;
using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;

namespace GithubRunnerRegistration
{
    public interface IGetCredentials
    {
        Task<RunnerRegistrationSecretData> GetCredentialsFromPath(string dir);
    }

    public class GetCredentials : IGetCredentials
    {
        public string Runner { get; private set; }
        public string Credentials { get; private set; }
        public string CredentialsRsaParams { get; private set; }
        public string PrivatePem { get; private set; }
        public string PublicPem { get; private set; }

        public async Task<RunnerRegistrationSecretData> GetCredentialsFromPath(string dir)
        {
            string json = await File.ReadAllTextAsync(Path.Combine(dir, ".credentials_rsaparams"));
            var rsaParams = JsonSerializer.Deserialize<RSAParametersSerializable>(json);
            var rsa = RSA.Create(rsaParams.RSAParameters);
            var privateKey = $"-----BEGIN PRIVATE KEY-----\n{ Convert.ToBase64String(rsa.ExportPkcs8PrivateKey(), Base64FormattingOptions.InsertLineBreaks)}\n-----END PRIVATE KEY-----\n";
            var publicKey = $"-----BEGIN PUBLIC KEY-----\n{ Convert.ToBase64String(rsa.ExportRSAPublicKey(), Base64FormattingOptions.InsertLineBreaks)}\n-----END PUBLIC KEY-----\n";
            var creds = await File.ReadAllBytesAsync(Path.Combine(dir, ".credentials"));
            var clientId = await this.GetClientId(creds);
            return new RunnerRegistrationSecretData
            {
                Id = clientId,
                Runner = Convert.ToBase64String(await File.ReadAllBytesAsync(Path.Combine(dir, ".runner"))),
                Credentials = Convert.ToBase64String(creds),
                CredentialsRsaParams = Convert.ToBase64String(await File.ReadAllBytesAsync(Path.Combine(dir, ".credentials_rsaparams"))),
                PrivatePem = Convert.ToBase64String(Encoding.UTF8.GetBytes(privateKey)),
                PublicPem = Convert.ToBase64String(Encoding.UTF8.GetBytes(publicKey))
            };
        }

        private async Task<Guid> GetClientId(byte[] creds)
        {
            using var ms = new MemoryStream(creds);
            var credsObj = await DeserializeAnonymousTypeAsync(ms, new { data = new { clientId = Guid.Empty } });
            return credsObj.data.clientId;
        }

        private static ValueTask<TValue> DeserializeAnonymousTypeAsync<TValue>(Stream stream, TValue anonymousTypeObject, JsonSerializerOptions options = default, CancellationToken cancellationToken = default)
            => JsonSerializer.DeserializeAsync<TValue>(stream, options, cancellationToken);
    }
}
