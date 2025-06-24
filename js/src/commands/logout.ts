import { removeConfig } from '@/lib/utils/auth';
import { green, red } from 'kolorist';


export default async function() {
	try {
		const isSuccess = removeConfig();
		if (isSuccess) {
			console.log(`${green('✔')} You have Successfully loged out`);
			process.exit(0);
		}
	} catch (error) {
		console.error(
			`${red('✖')} ${error instanceof Error ? error.message : 'An unknown error occurred'}`
		);
		process.exit(1);
	}
}



