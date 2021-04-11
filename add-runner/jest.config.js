// eslint-disable-next-line no-undef
module.exports = {
	globals: {
		'ts-jest': {
			tsconfigFile: 'tsconfig.json'
		}
	},
	moduleFileExtensions: [
		'ts',
		'js'
	],
	transform: {
		'^.+\\.(ts|tsx)$': './node_modules/ts-jest/preprocessor.js'
	},
	testMatch: [
		'**/*.test.(ts|js)'
	],
	testEnvironment: 'node'
};