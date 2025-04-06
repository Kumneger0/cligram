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
			const me = await client?.getMe();
			if (me?.firstName) {
				console.log(`${green('✔')} You have Successfully loged in`);
				process.exit(0);
			}
		})().catch((error) => {
			console.error(
				`${red('✖')} ${error instanceof Error ? error.message : 'An unknown error occurred'}`
			);
			process.exit(1);
		});
	}
);
