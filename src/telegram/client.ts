import { Api, TelegramClient } from 'telegram';
import {
	Channel,
	ChannelInfo,
	Dialog,
	MessagesResponse,
	TelegramUser,
	UserInfo
} from '../lib/types/index.js';

export let chatUsers: UserInfo[] = [];

async function getChannelInfo(client: TelegramClient, channelId: bigInt.BigInteger) {
	const channel = await client.getEntity(await client.getInputEntity(channelId));
	return channel;
}

export async function searchUsers(client: TelegramClient, query: string) {
	if (!client.connected) { await client.connect(); }
	const result = await client.invoke(
		new Api.contacts.Search({
			q: query,
			limit: 3
		})
	);
	const users = result.users
		.filter((user) => { return user.className === 'User' })
		.map((user) => {
			return {
				firstName: user.firstName ?? '',
				peerId: user.id,
				accessHash: user.accessHash as unknown as bigInt.BigInteger,
				isBot: user.bot ?? false,
				unreadCount: 0,
				lastSeen: null,
				isOnline: false
			} satisfies UserInfo;
		});

	const chats = result.chats
		.filter((chat) => { return chat.className === 'Chat' })
		.map((chat) => {
			return {
				title: chat.title || '',
			chatId: chat.id
			}
		});

	const channels = await Promise.all(
		result.results
			.filter((result) => { return result.className === 'PeerChannel' })
			.map(async (chann) => {
				const channel = (await getChannelInfo(client, chann.channelId)) as unknown as Channel;
				return {
					title: channel.title,
					username: channel.username,
					channelId: channel.id.toString(),
					accessHash: channel.accessHash.toString(),
					isCreator: channel.creator,
					isBroadcast: channel.broadcast,
					participantsCount: channel.participantsCount,
					unreadCount: 0
				} satisfies ChannelInfo;
			})
	);

	return { users, chats, channels };
}

export async function getUserChats<T extends Dialog['peer']['className']>(
	client: TelegramClient,
	type: T
): Promise<T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]> {
	if (!client.connected) { await client.connect(); }

	const result = (await client.invoke(
		new Api.messages.GetDialogs({
			offsetDate: 0,
			offsetId: 0,
			offsetPeer: new Api.InputPeerEmpty(),
			limit: 30000
		})
	)) as unknown as MessagesResponse;

	if (type === 'PeerChannel') {
		const channels = result.dialogs.filter((dialog) => { return dialog.peer.className === 'PeerChannel' });
		const channelsInfo = (
			await Promise.all(
				channels.map(async (chan) => {
					const channel = (await getChannelInfo(client, chan.peer.channelId)) as unknown as Channel;

					return {
						title: channel.title,
						username: channel.username,
						channelId: channel.id.toString(),
						accessHash: channel.accessHash.toString(),
						isCreator: channel.creator,
						isBroadcast: channel.broadcast,
						participantsCount: channel.participantsCount,
						unreadCount: 0
					} satisfies ChannelInfo;
				})
			)
		).filter(({ isBroadcast }) => { return isBroadcast }) as ChannelInfo[];
		return channelsInfo as unknown as Promise<T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]>;
	}
	if (type === 'PeerUser') {
		const userChats = result.dialogs.filter((dialog) => { return dialog.peer.className === 'PeerUser' });
		const users = await Promise.all(
			userChats.map(async ({ peer, unreadCount }) => {
				try {
					const user = (await getUserInfo(client, peer.userId)) as unknown as TelegramUser | null;
					if (!user) { return null; }
					const wasOnline = user.status?.wasOnline;
					const date = wasOnline ? new Date(wasOnline * 1000) : null;

					return {
						firstName: user.firstName,
						isBot: user.bot,
						peerId: peer.userId,
						accessHash: user.accessHash as unknown as bigInt.BigInteger,
						unreadCount: unreadCount,
						lastSeen: date,
						isOnline: user.status?.className === 'UserStatusOnline'
					};
				} catch (err) {
					return null;
				}
			})
		);

		chatUsers = users.filter((user): user is NonNullable<typeof user> => { return user !== null });
		return chatUsers.filter(({ isBot, firstName }) => { return !isBot && firstName }) as unknown as Promise<
			T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]
		>;
	}
	return [];
}

export async function getUserInfo(client: TelegramClient, userId: bigInt.BigInteger) {
	try {
		if (!client.connected) { await client.connect(); }
		const user = await client.getEntity(await client.getInputEntity(userId));
		return user;
	} catch (err) {
		return null
	}
}
