import { Container, getRandom } from "@cloudflare/containers";

export class PuasaSunnahContainer extends Container {
	defaultPort = 8080;
	sleepAfter = "2m";
}

export default {
	async fetch(request, env) {
		const container = await getRandom(env.API_CONTAINER, 3);
		return container.fetch(request);
	},
};
