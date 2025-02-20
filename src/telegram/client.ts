import { Api, TelegramClient } from 'telegram';
import { MessagesResponse, User } from '../lib/types/index.js';
import { ChatUser } from '../types.js';

export let chatUsers: ChatUser[] = [];

export async function getUserChats(client: TelegramClient) {
	if (!client.connected) await client.connect();

	const result = (await client.invoke(
		new Api.messages.GetDialogs({
			offsetDate: 0,
			offsetId: 0,
			offsetPeer: new Api.InputPeerEmpty(),
			limit: 30000
		})
	)) as unknown as MessagesResponse;
	const userChats = result.dialogs.filter((dialog) => dialog.peer.className === 'PeerUser');
	const users = await Promise.all(
		userChats.map(async ({ peer, unreadCount }) => {
			try {
				const user = (await getUserInfo(client, peer.userId)) as unknown as User;
				if (!user) return null;
				const wasOnline = user.status?.wasOnline
				const date = wasOnline ? new Date(wasOnline * 1000) : null

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
	return chatUsers.filter(({ isBot }) => !isBot);
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
