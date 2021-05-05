using GithubRunnerRegistration.Models;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Logging;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

// For more information on enabling Web API for empty projects, visit https://go.microsoft.com/fwlink/?LinkID=397860

namespace GithubRunnerRegistration.Api.Controllers
{
    [Produces("application/json")]
    [Route("api/[controller]")]
    [ApiController]
    public class RegisterController : ControllerBase
    {
        private readonly ILogger<RegisterController> logger;
        
        public RegisterController(ILogger<RegisterController> logger)
        {
            this.logger = logger;
        }


        /// <summary>
        /// Registers runners against the provided repo and returns the data required to associate each runner with the repo.
        /// The data returned can be used as the data property of a secret which is then referenced by ScaledActionRunner.RunnerSecrets
        /// The secret will then be mnoounted by a pod and used to register with Github
        /// </summary>
        /// <param name="request"></param>
        /// <returns>A map of secret names to secret data</returns>
        /// <response code="201">Request passed validation and runner(s) have been registered</response>
        /// <response code="202">Request passed validation but no runners were created because dryRun was specified, dummy information has been returned</response>
        /// <response code="400">Supplied request is invalid</response>  
        [HttpPost]
        [ProducesResponseType(StatusCodes.Status201Created)]
        [ProducesResponseType(StatusCodes.Status202Accepted)]
        [ProducesResponseType(StatusCodes.Status400BadRequest)]
        public async Task<ActionResult<Dictionary<string,RunnerRegistrationSecretData>>> Post([FromBody] RegistrationRequest request)
        {
            try
            {
                var runners = new Dictionary<string, RunnerRegistrationSecretData>();
                var register = new Register(request);
                await register.Setup();
                var identifier = new StringBuilder();
                identifier.AppendFormat ("runners://{0}/{1}", request.Owner, request.Repository);
                foreach (var name in request.RunnerNames)
                {
                    this.logger.LogInformation("Adding runner {name} to {Owner}/{Repository}", name, request.Owner, request.Repository);
                    var creds = await register.AddRunner(name, request.DryRun);
                    runners.Add(name, creds);
                    identifier.AppendFormat("/{0}", creds.Id);
                }
                // This is essentially a registration request ID
                var idUri = new Uri(identifier.ToString());
                if (request.DryRun)
                {
                    return this.Accepted(idUri, runners);
                }
                return this.Created(idUri,runners);
            }
            catch (InvalidOperationException ex)
            {
                this.logger.LogWarning(ex, "Error adding runners to {Owner}/{Repository}", request.Owner, request.Repository);
                return this.BadRequest(ex.Message);
            }
            catch (SetupException ex)
            {
                this.logger.LogWarning(ex, "Error adding runners to {Owner}/{Repository}", request.Owner, request.Repository);
                return this.BadRequest(ex.Message);
            }
            catch (Exception ex)
            {
                this.logger.LogError(ex, "Error adding runners to {Owner}/{Repository}", request.Owner, request.Repository);
                throw ex;
            }
        }
    }
}
