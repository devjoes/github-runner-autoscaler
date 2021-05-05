using System;
using System.Collections.Generic;
using System.Text;
using System.Text.Json.Serialization;
using System.Threading.Tasks;

namespace GithubRunnerRegistration.Models
{
    /// <summary>
    /// The data property of a secret containing all of the required data to register a GitHub actions runner.
    /// </summary>
    public class RunnerRegistrationSecretData
    {
        /// <summary>
        /// Client ID from .credentials
        /// </summary>
        [JsonIgnore]
        public Guid Id { get; set; }

        /// <summary>
        /// Base64 encoded general information about the runner
        /// </summary>
        [JsonPropertyName(".runner")]
        public string Runner { get; set; }

        /// <summary>
        /// Base64 encoded non-sensitive credential information (ClientID etc)
        /// </summary>
        [JsonPropertyName(".credentials")]
        public string Credentials { get; set; }

        /// <summary>
        /// Base64 encoded public and private keys (in a proprietry format)
        /// </summary>
        [JsonPropertyName(".credentials_rsaparams")]
        public string CredentialsRsaParams { get; set; }

        /// <summary>
        /// Base64 encoded private key from .credentials_rsaparams, not used currently.
        /// </summary>
        [JsonPropertyName("private.pem")]
        public string PrivatePem { get; set; }

        /// <summary>
        /// Base64 encoded public key from .credentials_rsaparams, not used currently.
        /// </summary>
        [JsonPropertyName("public.pem")]
        public string PublicPem { get; set; }
    }
}
