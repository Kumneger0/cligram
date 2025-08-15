import dotenv from 'dotenv';

dotenv.config();

export const getApiKeys = () => {
	const apiId = process.env.TELEGRAM_API_ID;
	const apiHash = process.env.TELEGRAM_API_HASH;

	if (!apiId || !apiHash) {
		throw new Error(
			'Missing required environment variables: TELEGRAM_API_ID and/or TELEGRAM_API_HASH. Please set them in your .env file.'
		);
	}

	return { apiId, apiHash };
};
