using GithubRunnerRegistration.Models;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Logging;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

// For more information on enabling Web API for empty projects, visit https://go.microsoft.com/fwlink/?LinkID=397860

namespace GithubRunnerRegistration.Api.Controllers
{
    [Route("api/[controller]")]
    [ApiController]
    public class RegisterController : ControllerBase
    {
        private readonly ILogger<RegisterController> logger;
        
        public RegisterController(ILogger<RegisterController> logger)
        {
            this.logger = logger;
        }
        [HttpPost]
        public async Task<ActionResult> Post([FromBody] RegistrationRequest request)
        {
            try
            {
                var runners = new Dictionary<string, Dictionary<string, string>>();
                var register = new Register(request);
                await register.Setup();
                foreach (var name in request.RunnerNames)
                {
                    this.logger.LogInformation("Adding runner {name} to {Owner}/{Repository}", name, request.Owner, request.Repository);
                    var creds = await register.AddRunner(name);
                    runners.Add(name, creds);
                }
                return this.Ok(runners);
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
