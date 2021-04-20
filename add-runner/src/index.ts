import fs from "fs";

import { GetConfig } from "./config";
import GetResources from "./getResources";

export async function main(): Promise<void> {
	const config = GetConfig();
	const resources = await GetResources(config);
	let out = process.stdout as NodeJS.WritableStream;
	if (config.output != "") {
		out = fs.createWriteStream(config.output);
		out.on("error", console.error);
	}

	for (let i = 0; i < resources.length; i++) {
		const yml = resources[i];
		out.write(yml);
		if (i < resources.length - 1) {
			out.write("\n---\n");
		}
	}
}
main();
