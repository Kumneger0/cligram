import { $ } from 'bun';
const compile = async () => {
	const root = `${process.cwd()}/src/index.ts`;

	// Compile for Linux x64
	await $`bun build --compile --target=bun-linux-x64 --minify --sourcemap  ${root} --outfile dist/tgcli-linux-x64`;
	// Compile for Linux ARM64
	await $`bun build --compile --target=bun-linux-arm64 --minify --sourcemap  ${root} --outfile dist/tgcli-linux-arm64`;
	// Compile for macOS x64
	await $`bun build --compile --target=bun-darwin-x64 --minify --sourcemap  ${root} --outfile dist/tgcli-darwin-x64`;
	// Compile for macOS ARM64
	await $`bun build --compile --target=bun-darwin-arm64 --minify --sourcemap ${root} --outfile dist/tgcli-darwin-x64`;
	// Compile for Windows x64
	await $`bun build --compile --target=bun-windows-x64 --minify --sourcemap ${root} --outfile dist/tgcli-win-x64`;
};
compile();
