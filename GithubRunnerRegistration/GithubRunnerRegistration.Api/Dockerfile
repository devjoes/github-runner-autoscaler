#See https://aka.ms/containerfastmode to understand how Visual Studio uses this Dockerfile to build your images for faster debugging.

FROM mcr.microsoft.com/dotnet/core/aspnet:3.1-buster-slim AS base
COPY --from=myoung34/github-runner:latest /actions-runner/ /actions-runner/
WORKDIR /app
EXPOSE 80

FROM mcr.microsoft.com/dotnet/core/sdk:3.1-buster AS build
WORKDIR /src
COPY ["GithubRunnerRegistration/GithubRunnerRegistration.csproj", "GithubRunnerRegistration/"]
RUN dotnet restore "GithubRunnerRegistration/GithubRunnerRegistration.csproj"
COPY . .
WORKDIR "/src/GithubRunnerRegistration"
RUN dotnet build "GithubRunnerRegistration.csproj" -c Release -o /app/build

FROM build AS publish
RUN dotnet publish "GithubRunnerRegistration.csproj" -c Release -o /app/publish

FROM base AS final
WORKDIR /app
COPY --from=publish /app/publish .
ENTRYPOINT ["dotnet", "GithubRunnerRegistration.dll"]