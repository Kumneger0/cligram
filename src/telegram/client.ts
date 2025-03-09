import { Api, TelegramClient } from 'telegram';
import { Dialog, MessagesResponse, User } from '../lib/types/index.js';
import { Channel, ChatUser } from '../types.js';

export let chatUsers: ChatUser[] = [];

export interface ChannelInfo {
	title: string;
	username: string | undefined;
	channelId: string;
	accessHash: string;
	isCreator: boolean;
	isBroadcast: boolean;
	participantsCount: number | null;
	unreadCount: number;
}

async function getChannelInfo(client: TelegramClient, channelId: bigInt.BigInteger) {
	const channel = await client.getEntity(await client.getInputEntity(channelId));
	return channel;
}

export async function getUserChats<T extends Dialog['peer']['className']>(
	client: TelegramClient,
	type: T
): Promise<T extends 'PeerChannel' ? ChannelInfo[] : ChatUser[]> {
	if (!client.connected) await client.connect();

	const result = (await client.invoke(
		new Api.messages.GetDialogs({
			offsetDate: 0,
			offsetId: 0,
			offsetPeer: new Api.InputPeerEmpty(),
			limit: 30000
		})
	)) as unknown as MessagesResponse;

	if (type === 'PeerChannel') {
		const channels = result.dialogs.filter((dialog) => dialog.peer.className === 'PeerChannel');
		const channelsInfo = (await Promise.all(
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
		)).filter(({ isBroadcast }) => isBroadcast) as ChannelInfo[];
		return channelsInfo as unknown as Promise<T extends 'PeerChannel' ? ChannelInfo[] : ChatUser[]>;
	}
	if (type === 'PeerUser') {
		const userChats = result.dialogs.filter((dialog) => dialog.peer.className === 'PeerUser');
		const users = await Promise.all(
			userChats.map(async ({ peer, unreadCount }) => {
				try {
					const user = (await getUserInfo(client, peer.userId)) as unknown as User;
					if (!user) return null;
					const wasOnline = user.status?.wasOnline;
					const date = wasOnline ? new Date(wasOnline * 1000) : null;

					return {
						firstName: user.firstName,
						isBot: user.bot,
						peerId: peer.userId,
						accessHash: user.accessHash as unknown as bigInt.BigInteger,
						unreadCount: unreadCount,
						lastSeen: date,
						isOnline: user.status?.className == 'UserStatusOnline'
					};
				} catch (err) {
					console.error(err);
					return null;
				}
			})
		);

		chatUsers = users.filter((user): user is NonNullable<typeof user> => user !== null);
		return chatUsers.filter(({ isBot }) => !isBot) as unknown as Promise<
			T extends 'PeerChannel' ? ChannelInfo[] : ChatUser[]
		>;
	}
	return [];
}

export async function getUserInfo(client: TelegramClient, userId: bigInt.BigInteger) {
	try {
		if (!client.connected) await client.connect();
		const user = await client.getEntity(await client.getInputEntity(userId));
		return user;
	} catch (err) {
		console.error(err);
	}
}
