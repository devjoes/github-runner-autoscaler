using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.ComponentModel.DataAnnotations;

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
        [Required]
        public string[] RunnerNames { get; set; }

        /// <summary>
        /// Token with repo:* rights of a user with admin access to repo
        /// </summary>
        [Required]
        public string AdminPat { get; set; }

        /// <summary>
        /// Github Owner
        /// </summary>
        [Required]
        public string Owner { get; set; }

        /// <summary>
        /// Github Repository
        /// </summary>
        [Required]
        public string Repository { get; set; }

        /// <summary>
        /// Runner labels to add to the runner (in addition to the standard 'self-hosted' etc)
        /// </summary>
        public string[] Labels { get; set; }

        /// <summary>
        /// If set the all of the validation steps will be carried out but the runner(s) will not actually be registered. Useful as a PR check.
        /// </summary>
        [DefaultValue(false)]
        public bool DryRun { get; set; }
    }
}
