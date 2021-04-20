using System;
using System.Collections.Generic;
using System.Text;

namespace GithubRunnerRegistration.Models
{
    public class SetupException : Exception
    {
        public SetupException(string message, Exception ex) : base(message, ex) { }
    }
}
