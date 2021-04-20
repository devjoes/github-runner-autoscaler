using System;
using System.Collections.Generic;

namespace GithubRunnerRegistration.Models
{
    /// <summary>
    /// Request to register n Github Action private runners
    /// </summary>
    public class RegistrationRequest
    {
        /// <summary>
        /// Names of runners to register
        /// </summary>
        public string[] RunnerNames { get; set; }

        /// <summary>
        /// Token with repo:* rights of a user with admin access to repo
        /// </summary>
        public string AdminPat { get; set; }

        /// <summary>
        /// Github Owner
        /// </summary>
        public string Owner { get; set; }

        /// <summary>
        /// Github Repository
        /// </summary>
        public string Repository { get; set; }

        /// <summary>
        /// Runner labels to add to the runner (in addition to the standard 'self-hosted' etc)
        /// </summary>
        public string[] Labels { get; set; }
    }
}
