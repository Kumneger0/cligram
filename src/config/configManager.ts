import fs from 'fs';
import os from 'os';
import path from 'path';
import { type CliGramConfigSchema, DEFAULT_CONFIG, cliGramConfigSchema } from './types';

const CONFIG_PATH = path.join(os.homedir(), '.cligram', 'user.config.json');

export const loadConfig = (): CliGramConfigSchema => {
	try {
		const configDir = path.dirname(CONFIG_PATH);
		if (!fs.existsSync(configDir)) {
			fs.mkdirSync(configDir, { recursive: true });
		}
		if (fs.existsSync(CONFIG_PATH)) {
			const fileContent = fs.readFileSync(CONFIG_PATH, 'utf-8');
			try {
                const parsedConfig = cliGramConfigSchema.parse(
                    JSON.parse(fileContent) as Partial<CliGramConfigSchema>
				);
				return { ...DEFAULT_CONFIG, ...parsedConfig };
			} catch (error) {
				if (error instanceof Error) {
					console.error('Invalid config:', error.message);
					process.exit(1);
				}
				return DEFAULT_CONFIG;
			}
		}
		return DEFAULT_CONFIG;
	} catch (error) {
		console.error('Error loading config:', error);
		return DEFAULT_CONFIG;
	}
};

let config = loadConfig();

export const getConfig = <K extends keyof CliGramConfigSchema>(key: K): CliGramConfigSchema[K] => {
	return config[key];
};

export const getAllConfig = (): CliGramConfigSchema => {
	return config;
};
