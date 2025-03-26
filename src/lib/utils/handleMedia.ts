import { Api, TelegramClient } from 'telegram';
import { getTelegramClient } from './auth';
import { Media, Message, MessageMediaPhoto } from '../types';

type MediaSize = 'large' | 'small';

type DownloadMediaArgs = {
	media: MessageMediaPhoto | Media | null;
	size: MediaSize;
};

export const downloadMedia = async ({ media, size }: DownloadMediaArgs): Promise<Buffer | null> => {
	if (!media) {
		throw new Error('Media is required');
	}
	const client = await getTelegramClient();
	if (!client) {
		throw new Error('Client is not connected');
	}
	try {
		if (!client.connected) {
			await client.connect();
		}
		return await handleMediaDownload(client, media, size);
	} catch (err) {
		if (err instanceof Error) {
			console.error(err.message);
		}
	} finally {
		await client.disconnect();
	}
	return null;
};
export const handleVideoDownload = async (): Promise<unknown> => {
	// TODO: Implement video download functionality
	return null;
};
export const handleMediaDownload = async (
	client: TelegramClient,
	media: Message['media'] | MessageMediaPhoto | Media,
	size: MediaSize
): Promise<Buffer | null> => {
	const buffer = await client.downloadMedia(media as unknown as Api.TypeMessageMedia, {
		progressCallback: (_progress, _total) => {},
		thumb: size === 'small' ? 0 : undefined
	});
	return buffer as Buffer;
};
