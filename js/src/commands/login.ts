import { authenticateUser } from '@/lib/utils/auth';
import { green, red } from 'kolorist';

export async function login () {
	try {
		const client = await authenticateUser({ isCalledFromLogin: true });
		const me = await client?.getMe();
		if (me?.firstName) {
			console.log(`${green('✔')} You have Successfully loged in`);
			process.exit(0);
		}
	} catch (error) {
		console.error(
			`${red('✖')} ${error instanceof Error ? error.message : 'An unknown error occurred'}`
		);
		process.exit(1);
	}
}
