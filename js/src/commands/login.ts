import { authenticateUser } from '@/lib/utils/auth';
import { green, red } from 'kolorist';

export async function login() {
	try {
		const client = await authenticateUser({ isCalledFromLogin: true });
		const me = await client?.getMe();
		if (me?.firstName) {
			const successMessage = `${green('✔')} You have Successfully loged`;
			process.stdout.write(successMessage);
			process.exit(0);
		}
	} catch (error) {
		const errMessage = `${red('✖')} ${error instanceof Error ? error.message : 'An unknown error occurred'}`;
		process.stdout.write(errMessage);
		process.exit(1);
	}
}
