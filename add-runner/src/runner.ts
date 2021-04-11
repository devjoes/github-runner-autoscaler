import Docker from "dockerode";
import tmp from "tmp";
import path from "path";
import fs from "fs";

export type RunnerCreds = {
	credentialsRsaparams: string;
	credentials: string;
	runner: string;
	privateKey: string;
	publicKey: string;
};

export default class Runner {
	image = "joeshearn/action-runner-sideloaded-config";
	runnerToken: string;
	docker = new Docker({ socketPath: "/var/run/docker.sock" });

	constructor(token: string) {
		this.runnerToken = token;
	}
	async setup(): Promise<void> {
		await this.docker.pull(this.image);
	}
	async addRunner(
		owner: string,
		repo: string,
		runnerName: string,
		labels?: string
	): Promise<RunnerCreds> {
		const output = tmp.dirSync({ unsafeCleanup: true });
		const overwriteOutputUid = process.platform === "linux" ? process.getuid() : 0;
		const data = await this.docker.run(this.image, [], process.stderr, {
			Volumes: {
				"/var/run/docker.sock": {},
				"/config_output": {},
			},
			Env: [
				`REPO_URL=https://github.com/${owner}/${repo}`,
				`RUNNER_NAME=${runnerName}`,
				`RUNNER_TOKEN=${this.runnerToken}`,
				"RETURN_CONFIG=1",
				`OVERWRITE_OUTPUT_UID=${overwriteOutputUid || ""}`,
			].concat(labels ? [`LABELS=${labels}`] : []),
			Hostconfig: {
				Binds: ["/var/run/docker.sock:/var/run/docker.sock", `${output.name}:/config_output`],
			},
		});
		const container = data[1];
		container.remove();

		const creds = getRunnerCreds(output.name);
		output.removeCallback();
		return creds;
	}
}

async function getRunnerCreds(outputPath: string): Promise<RunnerCreds> {
	const getCred = (fileName: string) => {
		const p = path.join(outputPath, fileName);
		if (!fs.existsSync(p)) {
			throw new Error("Missing credential output");
		}
		return fs.readFileSync(p).toString();
	};
	const creds = {
		credentialsRsaparams: getCred(".credentials_rsaparams"),
		credentials: getCred(".credentials"),
		runner: getCred(".runner"),
		privateKey: getCred("private.pem"),
		publicKey: getCred("public.pem"),
	};
	return creds;
}
