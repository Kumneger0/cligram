import dotenv from 'dotenv';

dotenv.config();

export const getApiKeys = () => {
  const apiId = process.env.TELEGRAM_API_ID;
  const apiHash = process.env.TELEGRAM_API_HASH;

  if (!apiId || !apiHash) {
    return { apiId: null, apiHash: null, error: 'Missing keys' };
  }

  return { apiId, apiHash };
};
