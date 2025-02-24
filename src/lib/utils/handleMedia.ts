import { Api, TelegramClient } from 'telegram';
import Message, { MessageMediaPhoto } from '../../types';
import { getTelegramClient } from './auth';

type MediaSize = 'large' | 'small';

type DownloadMediaArgs = {
	media: Message['media'] | MessageMediaPhoto;
	size: MediaSize;
};

export const downloadMedia = async ({ media, size }: DownloadMediaArgs): Promise<Buffer | null> => {
	if (!media) throw new Error('Media is required');
	const client = await getTelegramClient();
	if (!client) throw new Error('Client is not connected');
	try {
		if (!client?.connected) await client.connect();
		if (media) return await handleMediaDownload(client, media, size);
	} catch (err) {
		console.error(err);
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
	media: Message['media'] | MessageMediaPhoto,
	size: MediaSize
): Promise<Buffer | null> => {
	const buffer = await client.downloadMedia(media as unknown as Api.TypeMessageMedia, {
		progressCallback: (progress, total) => {
			const percent = (Number(progress) / Number(total)) * 100;
		},
		thumb: size === 'small' ? 0 : undefined
	});
	return buffer as Buffer;
};

export const downloadVideoThumbnail = async (client: TelegramClient, media: Message['media']) => {
	if (!client.connected) await client.connect();
	const thumbnail = media.document.thumbs;
	if (!thumbnail) return;
	const buffer = await client.downloadMedia(media as unknown as Api.TypeMessageMedia, {
		thumb: 1
	});
	if (!buffer) return;
	return buffer;
};
