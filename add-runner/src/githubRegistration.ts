import { Octokit } from "@octokit/rest";

export async function getRegToken(ghToken: string, owner: string, repo: string): Promise<string> {
	const github = new Octokit({ auth: ghToken });
	const regToken = await github.actions.createRegistrationTokenForRepo({
		owner,
		repo,
	});
	return regToken.data.token;
}
