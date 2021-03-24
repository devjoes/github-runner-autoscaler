FROM mcr.microsoft.com/dotnet/core/sdk:3.1-buster AS build
RUN dotnet new console -n RsaParamsToPem
COPY ./RsaParamsToPem.cs ./RsaParamsToPem/Program.cs
RUN dotnet publish -c release --self-contained --runtime linux-x64 RsaParamsToPem

FROM myoung34/github-runner:latest
COPY --from=build /RsaParamsToPem/bin/release/netcoreapp3.1/linux-x64/publish /RsaParamsToPem
COPY ./entrypoint.sh .
RUN chmod +x ./entrypoint.sh
ENTRYPOINT ./entrypoint.sh
