#See https://aka.ms/containerfastmode to understand how Visual Studio uses this Dockerfile to build your images for faster debugging.

FROM mcr.microsoft.com/dotnet/core/aspnet:3.1-buster-slim AS base
COPY --from=myoung34/github-runner:latest /actions-runner/ /actions-runner/
WORKDIR /app
ENV ASPNETCORE_URLS=http://+:8080
RUN chown 33 /actions-runner/ -R
USER 33
EXPOSE 8080

FROM mcr.microsoft.com/dotnet/core/sdk:3.1-buster AS build
WORKDIR /src
COPY ["GithubRunnerRegistration.Api/GithubRunnerRegistration.Api.csproj", "GithubRunnerRegistration.Api/"]
COPY ["GithubRunnerRegistration/GithubRunnerRegistration.csproj", "GithubRunnerRegistration/"]
COPY ["GithubRunnerRegistration.Tests/GithubRunnerRegistration.Tests.csproj", "GithubRunnerRegistration.Tests/"]
RUN dotnet restore "GithubRunnerRegistration.Api/GithubRunnerRegistration.Api.csproj"
RUN dotnet restore "GithubRunnerRegistration.Tests/GithubRunnerRegistration.Tests.csproj"
COPY . .
WORKDIR "/src/"
RUN dotnet test "GithubRunnerRegistration.Tests/GithubRunnerRegistration.Tests.csproj"
RUN dotnet build "GithubRunnerRegistration.Api/GithubRunnerRegistration.Api.csproj" -c Release -o /app/build

FROM build AS publish
RUN dotnet publish "GithubRunnerRegistration.Api/GithubRunnerRegistration.Api.csproj" -c Release -o /app/publish

FROM base AS final
WORKDIR /app
COPY --from=publish /app/publish .
ENTRYPOINT ["dotnet", "GithubRunnerRegistration.Api.dll"]