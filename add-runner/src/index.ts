import fs from "fs";
import Runner, { RunnerCreds } from "./runner";
import { getRegToken } from "./githubRegistration";
import getResources from "./resources";
import { Config, GetConfig } from "./config";

export async function main(): Promise<void> {
	const config = GetConfig();
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

	let out = process.stdout as NodeJS.WritableStream;
	if (config.output != "") {
		out = fs.createWriteStream(config.output);
		out.on("error", console.error);
	}

	const resources = getResources(config, creds);
	for (let i = 0; i < resources.length; i++) {
		const yml = resources[i];
		out.write(yml);
		if (i < resources.length - 1) {
			out.write("\n---\n");
		}
	}
}
main();
