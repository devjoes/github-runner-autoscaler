using GithubRunnerRegistration.Models;
using System;
using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;

namespace GithubRunnerRegistration
{
    public interface IGetCredentials
    {
        Task<Dictionary<string,string>> GetCredentialsFromPath(string dir);
    }

    public class GetCredentials : IGetCredentials
    {
        public async Task<Dictionary<string, string>> GetCredentialsFromPath(string dir)
        {
            string json = await File.ReadAllTextAsync(Path.Combine(dir, ".credentials_rsaparams"));
            var rsaParams = JsonSerializer.Deserialize<RSAParametersSerializable>(json);
            var rsa = RSA.Create(rsaParams.RSAParameters);
            var privateKey = $"-----BEGIN PRIVATE KEY-----\n{ Convert.ToBase64String(rsa.ExportPkcs8PrivateKey(), Base64FormattingOptions.InsertLineBreaks)}\n-----END PRIVATE KEY-----\n";
            var publicKey = $"-----BEGIN PUBLIC KEY-----\n{ Convert.ToBase64String(rsa.ExportRSAPublicKey(), Base64FormattingOptions.InsertLineBreaks)}\n-----END PUBLIC KEY-----\n";
            return new Dictionary<string, string>
            {
                {".runner",Convert.ToBase64String(await File.ReadAllBytesAsync(Path.Combine(dir, ".runner")))},
                {".credentials",Convert.ToBase64String(await File.ReadAllBytesAsync(Path.Combine(dir, ".credentials")))},
                {".credentials_rsaparams",Convert.ToBase64String(await File.ReadAllBytesAsync(Path.Combine(dir, ".credentials_rsaparams")))},
                {"private.pem",Convert.ToBase64String(Encoding.UTF8.GetBytes(privateKey))},
                {"public.pem",Convert.ToBase64String(Encoding.UTF8.GetBytes(publicKey))},
            };
        }
    }
}
