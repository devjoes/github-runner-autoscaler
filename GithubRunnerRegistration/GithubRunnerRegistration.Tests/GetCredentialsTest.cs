using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Threading.Tasks;
using Xunit;

namespace GithubRunnerRegistration.Tests
{
    public class GetCredentialsTest
    {
        const string runner = "runner content";
        const string credentials = "credentials content";
        // This is just a random RSA params object and is not sensitive
        const string credentialsRsaParams = "{\"RSAParameters\":{},\"d\":\"hXdVeIRBXrT8GgOF9N5rB0PGhEgH0x9Nm0bzhCbL3YvWxuium+q2Sc8wy9xwBflknLCGD9j5uCyOiD9uxfh3q5Im6sESV/VgetRuGqjNGhSaAnobX6KBVEwYtWnWzV14u0os1F6P0Nkij5cnAOHe1y7LuibdvVkcmqsGudZcUYUlmNitqknWAoEoxtNpu5BjvbC/8MuOZ488NJJBDuZSg75yJsY6KSk/wMXWVWthQ+IiJQqZhi15iJefrlpW5aCmqtbkY2KGKBij+CjvSdMPFrfiKCDZv8PxlzcIIaD9T9pM0p89Ti78p4jBlMsN3dpWjXIQvE3QMnegUeILrxx+RQ==\",\"dp\":\"B3D+bJdztNWoIAfCw8Q/UlsANmCkP7E6XhGydpAtOKN9UhJeK4E9NRRIs6p0glaRR6nsGN4KkM3Yl61ROr8Kn89IsUCZXyXbLnXOmi4BcMrSqssguDQfVlE3NLXOtTYDDv6pDdY2ZMSpThl4edMUydlF7F6bmDaFjMZoEk1SpL0=\",\"dq\":\"AbhSl9H9SVjYzTwg6fLqzB3GCdeC1KTtIM2Uokj/VxjWXIbAxFRG4uSCx3ddoCyRex6WV8nqYSpvsyCoX76+jzomYPdns5SL9kgB2BgIYPKRszUoY8PcNDpPHAPj1PlmzZFpNzsE++kmon54kyavwb47Nuo1A2KnSkw359T+pDU=\",\"exponent\":\"AQAB\",\"inverseQ\":\"kBH3mldmzKxoEdNsMHSPCXenFQySm1hyv3DimLwlDTGX835qp6+FPdFVZK8o2d86kN9xwe/iX+mDe0yK2vCHskgscJlMcGvP0+IgTDDW4ctkmHlD/X2lcJKUif96pmr10bpMm9ayN6Hc4BJFo9YTHtqfZ+Xfc950qPp9GfMtLYs=\",\"modulus\":\"omMMkenoxhTbvMvPsjrYsvWE+VDPMPi8mKgpOr+ngf9AKMGD1fldxsakuxtq1g4qZM768yFMsGPf/XsfHNIwEXXVhn6RSuUu22n0Y8fmHxSraceYXPOCi0mBC64HHZr6jWeCB5KiL2g8iGiNDti6Tsv3xeDU8ACA6LmCz37CB68VUsxXaQQLbGXo8+3qcQ53D5FwLafGSk4yl7gucF5SThvIDxOMsoE65TOxyhEMqiAnGTCU50XJURPj1GBXCzW17CjDmsQ64QKOlNm9EJzmMFo6ttPoOxZRb8usJps0nzPcq2IVhihCnX+udlAttkmo0fdFHKhEKzFbgSvX0vVMWQ==\",\"p\":\"yVtPBKM6ljwjRR1Bsa3h1YsohbE3NaOgX/r7lpTMt5EDVgFLPkyZt9xAQkoBxmce42CVHghlWQqBiPmK5MI0ujrSLxD6Q/zdJzZw4kHqT4M4MM+J+HqPlYPcRC0tWKabsYrkXBdHIe9OuDUfACoh6+yHC0KV91hWmQu3WtYIQDs=\",\"q\":\"znRs1/NkBqBr4I2qs/b4+5Q71ZdKJ0VJBBVNxMAPDEYTZNwI8mxoouHWuq0N2aTg6Ja46K2PCrKTBNsx43Rr01JV21PFYpPyUoF3I5Nx7MCJvdLM24u1Bd3uVc1KfFvLAQm+qObILCLzpJ76MkqBm6hNbmF9du9wCSWav7gEUHs=\"}";

        [Fact]
        public async Task ShouldParseCredentials()
        {
            var tmp = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
            Directory.CreateDirectory(tmp);
            try
            {
                File.WriteAllText(Path.Combine(tmp, ".runner"), runner);
                File.WriteAllText(Path.Combine(tmp, ".credentials"), credentials);
                File.WriteAllText(Path.Combine(tmp, ".credentials_rsaparams"), credentialsRsaParams);
                var getCreds = new GetCredentials();
                var creds = await getCreds.GetCredentialsFromPath(tmp);
                Assert.Equal(5, creds.Count);
                Assert.Equal(Convert.ToBase64String(Encoding.UTF8.GetBytes(credentialsRsaParams)), creds[".credentials_rsaparams"]);
                Assert.Equal(Convert.ToBase64String(Encoding.UTF8.GetBytes(runner)), creds[".runner"]);
                Assert.Equal(Convert.ToBase64String(Encoding.UTF8.GetBytes(credentials)), creds[".credentials"]);
                Assert.NotNull(creds["private.pem"]);
                Assert.NotNull(creds["public.pem"]);
            }
            finally
            {
                Directory.Delete(tmp, true);
            }
        }
    }
}
