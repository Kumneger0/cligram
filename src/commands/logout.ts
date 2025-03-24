import { removeConfig } from '@/lib/utils/auth';
import { command } from 'cleye';
import { red, green } from 'kolorist';

export default command(
	{
		name: 'logout',
		help: {
			description: 'logout'
		}
	},
	(_argv) => {
		(async () => {
			const isSuccess = removeConfig();
			if (isSuccess) {
				console.log(`${green('✔')} You have Successfully loged out`);
				process.exit(0);
			}
		})().catch((error) => {
			console.error(`${red('✖')} ${error instanceof Error ? error.message : 'An unknown error occurred'}`);
			process.exit(1);
		});
	}
);
