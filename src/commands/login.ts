import { getTelegramClient } from '../lib/utils/auth';
import { command } from 'cleye';
import { green, red } from 'kolorist';

export default command(
	{
		name: 'login',
		help: {
			description: 'login'
		}
	},
	(_argv) => {
		(async () => {
			const client = await getTelegramClient(true);
			if (client) {
				console.log(`${green('✔')} You have Successfully loged in`);
				process.exit(0);
			}
		})().catch((error) => {
			console.error(`${red('✖')} ${error.message}`);
			process.exit(1);
		});
	}
);
