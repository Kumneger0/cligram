import dotenv from 'dotenv';

dotenv.config();

export const getApiKeys = () => {
  const apiId = process.env.TELEGRAM_API_ID;
  const apiHash = process.env.TELEGRAM_API_HASH;

  if (!apiId || !apiHash) {
    throw new Error('Error: TELEGRAM_API_ID and TELEGRAM_API_HASH must be set in your environment during build.');
  }

  return { apiId, apiHash };
};
