import fs from "fs";
import { parse } from "ts-command-line-args";

export type Config = {
	name: string;
	readPat: string;
	adminPat: string;
	maxRunners: number;
	owner: string;
	repo: string;
	labels?: string;
	githubNs: string;
	output: string;
	help: boolean;
};

export function GetConfig(): Config {
	const config = parse<Config>(
		{
			name: {
				type: String,
				alias: "n",
				description: "Name to be used for Stateful Set, HPA and Scaled Object",
			},
			readPat: {
				type: String,
				alias: "p",
				description:
					"PAT token of account with read access to repo. Can also be a path to a file containing the token.",
			},
			adminPat: {
				type: String,
				alias: "a",
				description:
					"PAT token of account with admin access to repo (can be removed after setup). Can also be a path to a file containing the token.",
			},
			output: {
				type: String,
				alias: "f",
				defaultValue: "",
				description: "Write output to file.",
			},
			maxRunners: { type: Number, alias: "m", description: "Maximum number of runners" },
			owner: { type: String, alias: "o", description: "Repo owner" },
			repo: { type: String, alias: "r", description: "Repo name" },
			labels: { type: String, optional: true, alias: "l", description: "Labels to add to runner" },
			githubNs: { type: String, alias: "g", description: "Github API server namespace" },
			help: {
				type: Boolean,
				defaultValue: false,
				alias: "h",
				description: "Prints this usage guide",
			},
		},
		{
			helpArg: "help",
			headerContentSections: [
				{
					header: "Add Runner",
					content:
						"Creates the required secrets and ScaledActionRunner to set up auto scaling Github runners",
				},
			],
			footerContentSections: [{ content: `https://github.com/devjoes/github-runner-autoscaler` }],
		}
	);
	const parsePat = (t: string) => {
		const rx = /^[a-z0-9_]{40}$/i;
		let token = t.trim();
		if (!token.match(rx) && fs.existsSync(token)) {
			token = fs.readFileSync(token).toString().trim();
		}
		if (token.match(rx)) {
			return token;
		}
		throw new Error(`${token} as not a valid PAT token (${rx}) and does not exist as a file.`);
	};
	config.adminPat = parsePat(config.adminPat);
	config.readPat = parsePat(config.readPat);
	return config;
}
