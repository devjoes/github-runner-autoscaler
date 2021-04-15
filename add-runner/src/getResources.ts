import Runner, { RunnerCreds } from "./runner";
import { getRegToken } from "./githubRegistration";
import generateResources from "./resources";
import { Config } from "./config";

export default async function (config: Config): Promise<string[]> {
	const token = await getRegToken(config.adminPat, config.owner, config.repo);
	const runners = new Runner(token);
	await runners.setup();
	const creds = [] as Array<RunnerCreds>;
	for (let i = 0; i < config.maxRunners; i++) {
		const c = await runners.addRunner(
			config.owner,
			config.repo,
			`${config.name}-${i}`,
			config.labels
		);
		creds.push(c);
	}

	return generateResources(config, creds);
}
