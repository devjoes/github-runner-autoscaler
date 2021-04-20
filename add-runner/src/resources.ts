import YAML from "yaml";
import { Config } from "./config";
import { RunnerCreds } from "./runner";

const btoa = (input: string): string => Buffer.from(input).toString("base64");
function generateSecret(name: string, namespace: string, data: { [key: string]: string }): Secret {
	return {
		apiVersion: "v1",
		kind: "Secret",
		metadata: {
			name,
			namespace,
		},
		data,
		type: "Opaque",
	};
}
function generateRunnerSecret(name: string, namespace: string, creds: RunnerCreds): Secret {
	return generateSecret(name, namespace, {
		".credentials_rsaparams": btoa(creds.credentialsRsaparams),
		".credentials": btoa(creds.credentials),
		".runner": btoa(creds.runner),
		"private.pem": btoa(creds.privateKey),
		"public.pem": btoa(creds.publicKey),
	});
}
function generateScaledActionRunner(config: Config): ScaledActionRunner {
	const runner = {
		kind: "ScaledActionRunner",
		apiVersion: "runner.devjoes.com/v1alpha1",
		metadata: {
			name: config.name,
		},
		spec: {
			githubTokenSecret: config.name,
			maxRunners: config.maxRunners,
			owner: config.owner,
			repo: config.repo,
			runnerSecrets: [],
		},
	} as ScaledActionRunner;
	for (let i = 0; i < config.maxRunners; i++) {
		runner.spec.runnerSecrets.push(`${config.name}-${i}`);
	}
	if (config.labels) {
		runner.spec.runner = { labels: config.labels };
		runner.spec.selector = config.labels
			.split(/,/g)
			.map((l) => "wf_runs_on_" + l.replace(/[^a-z0-9-]/gi, "_"))
			.join(",");
	}
	return runner;
}

export default function (config: Config, creds: Array<RunnerCreds>): Array<string> {
	const sar = generateScaledActionRunner(config);
	const githubPatNs = config.githubPatNs ? config.githubPatNs : config.statefulSetNs;
	const readPat = generateSecret(config.name, githubPatNs, { token: btoa(config.readPat) });
	const runnerSecrets = creds.map((c, i) =>
		generateRunnerSecret(`${config.name}-${i}`, config.statefulSetNs, c)
	);
	return [
		YAML.stringify(sar),
		YAML.stringify(readPat),
		...runnerSecrets.map((s) => YAML.stringify(s)),
	];
}

type ScaledActionRunner = {
	apiVersion: string;
	kind: string;
	metadata: { name: string };
	spec: ScaledActionRunnerSpec;
};
type Runner = {
	labels: string;
};
type ScaledActionRunnerSpec = {
	githubTokenSecret: string;
	maxRunners: number;
	owner: string;
	repo: string;
	runner?: Runner;
	runnerSecrets: string[];
	selector?: string;
};
type Secret = {
	apiVersion: string;
	kind: string;
	metadata: { name: string; namespace?: string };
	data: { [key: string]: string };
	type: string;
};
