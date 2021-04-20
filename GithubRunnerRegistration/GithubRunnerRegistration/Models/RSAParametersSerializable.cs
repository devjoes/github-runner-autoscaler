using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json.Serialization;

namespace GithubRunnerRegistration.Models
{
    [Serializable]
    public class RSAParametersSerializable : ISerializable
    {
        private RSAParameters _rsaParameters;

        public RSAParameters RSAParameters
        {
            get
            {
                return _rsaParameters;
            }
        }

        public RSAParametersSerializable(RSAParameters rsaParameters)
        {
            _rsaParameters = rsaParameters;
        }

        public RSAParametersSerializable()
        {
        }

        [JsonPropertyName("d")]
        public byte[] D { get { return _rsaParameters.D; } set { _rsaParameters.D = value; } }

        [JsonPropertyName("dp")]
        public byte[] DP { get { return _rsaParameters.DP; } set { _rsaParameters.DP = value; } }

        [JsonPropertyName("dq")]
        public byte[] DQ { get { return _rsaParameters.DQ; } set { _rsaParameters.DQ = value; } }

        [JsonPropertyName("exponent")]
        public byte[] Exponent { get { return _rsaParameters.Exponent; } set { _rsaParameters.Exponent = value; } }

        [JsonPropertyName("inverseQ")]
        public byte[] InverseQ { get { return _rsaParameters.InverseQ; } set { _rsaParameters.InverseQ = value; } }

        [JsonPropertyName("modulus")]
        public byte[] Modulus { get { return _rsaParameters.Modulus; } set { _rsaParameters.Modulus = value; } }

        [JsonPropertyName("p")]
        public byte[] P { get { return _rsaParameters.P; } set { _rsaParameters.P = value; } }

        [JsonPropertyName("q")]
        public byte[] Q { get { return _rsaParameters.Q; } set { _rsaParameters.Q = value; } }

        public RSAParametersSerializable(SerializationInfo information, StreamingContext context)
        {
            _rsaParameters = new RSAParameters()
            {
                D = (byte[])information.GetValue("d", typeof(byte[])),
                DP = (byte[])information.GetValue("dp", typeof(byte[])),
                DQ = (byte[])information.GetValue("dq", typeof(byte[])),
                Exponent = (byte[])information.GetValue("exponent", typeof(byte[])),
                InverseQ = (byte[])information.GetValue("inverseQ", typeof(byte[])),
                Modulus = (byte[])information.GetValue("modulus", typeof(byte[])),
                P = (byte[])information.GetValue("p", typeof(byte[])),
                Q = (byte[])information.GetValue("q", typeof(byte[]))
            };
        }

        public void GetObjectData(SerializationInfo info, StreamingContext context)
        {
            info.AddValue("d", _rsaParameters.D);
            info.AddValue("dp", _rsaParameters.DP);
            info.AddValue("dq", _rsaParameters.DQ);
            info.AddValue("exponent", _rsaParameters.Exponent);
            info.AddValue("inverseQ", _rsaParameters.InverseQ);
            info.AddValue("modulus", _rsaParameters.Modulus);
            info.AddValue("p", _rsaParameters.P);
            info.AddValue("q", _rsaParameters.Q);
        }
    }
}
