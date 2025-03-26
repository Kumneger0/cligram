import { Api, TelegramClient } from 'telegram';
import {
	Channel,
	ChannelInfo,
	Dialog,
	MessagesResponse,
	TelegramUser,
	UserInfo
} from '../lib/types/index.js';
import { getConfig } from '@/config/configManager.js';

export let chatUsers: UserInfo[] = [];

const lastSeenMessages = {
	UserStatusRecently: 'last seen recently',
	UserStatusLastMonth: 'last seen within a month',
	UserStatusLastWeek: 'last seen within a week'
};

/**
 * Updates the user's online status on Telegram.
 * 
 * @param {TelegramClient} client - The Telegram client instance
 * @param {boolean} online - Whether to set the user as online (true) or offline (false)
 * @returns {Promise<void>} A promise that resolves when the status is updated
 * 
 * @description
 * This function updates whether the user appears online or offline to other users.
 * The visibility of this status depends on the user's privacy settings.
 */
export const setUserOnlineStatus = async (client: TelegramClient, online: boolean) => {
	try {
		await client.invoke(
			new Api.account.UpdateStatus({
				offline: !online,
			})
		);
	} catch (err) {
		console.error(err);
	}
};



/**
 * Sets the user's privacy settings for last seen visibility on Telegram.
 *
 * @param {TelegramClient} client - The Telegram client instance
 * @returns {Promise<void>} A promise that resolves when privacy settings are updated
 *
 * @description
 * This function updates the user's "last seen" privacy settings based on their config.
 * The lastSeenVisibility can be set to:
 * - 'everyone': Allow all users to see last seen status
 * - 'contacts': Only allow contacts to see last seen status
 * - 'nobody': Don't allow anyone to see last seen status
 */
export async function setUserPrivacy(client: TelegramClient) {
	try {
		const config = getConfig('privacy');
		const rules = {
			everyone: new Api.InputPrivacyValueAllowAll(),
			contacts: new Api.InputPrivacyValueAllowContacts(),
			nobody: new Api.InputPrivacyValueDisallowAll()
		} as const;

		if (config.lastSeenVisibility) {
			const rule = rules[config.lastSeenVisibility];
			await client.invoke(
				new Api.account.SetPrivacy({
					key: new Api.InputPrivacyKeyStatusTimestamp(),
					rules: [rule]
				})
			);
		}
	} catch (err) {
		console.error(err);
	}
}

async function getChannelInfo(client: TelegramClient, channelId: bigInt.BigInteger) {
	const channel = await client.getEntity(await client.getInputEntity(channelId));
	return channel;
}

/**
 * Searches for users, chats and channels on Telegram based on a query string.
 *
 * @param {TelegramClient} client - The Telegram client instance
 * @param {string} query - The search query string
 * @returns {Promise<{users: UserInfo[], chats: {title: string, chatId: number}[], channels: ChannelInfo[]}>}
 * A promise that resolves to an object containing:
 * - users: Array of matching user info
 * - chats: Array of matching chat info
 * - channels: Array of matching channel info
 *
 * @description
 * This function searches across Telegram entities and returns:
 * - Up to 3 matching users with their basic info
 * - Matching group chats with title and ID
 * - Matching channels with full channel information
 */
export async function searchUsers(client: TelegramClient, query: string) {
	if (!client.connected) {
		await client.connect();
	}
	const result = await client.invoke(
		new Api.contacts.Search({
			q: query,
			limit: 3
		})
	);
	const users = result.users
		.filter((user) => {
			return user.className === 'User';
		})
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
		.filter((chat) => {
			return chat.className === 'Chat';
		})
		.map((chat) => {
			return {
				title: chat.title || '',
				chatId: chat.id
			};
		});

	const channels = await Promise.all(
		result.results
			.filter((result) => {
				return result.className === 'PeerChannel';
			})
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

/**
 * Retrieves user chats or channels from Telegram based on the specified type.
 *
 * @template T - The type of peer to retrieve ('PeerChannel' or 'PeerUser')
 * @param {TelegramClient} client - The Telegram client instance
 * @param {T} type - The type of peers to retrieve ('PeerChannel' or 'PeerUser')
 * @returns {Promise<T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]>} A promise that resolves to:
 *   - An array of ChannelInfo objects if type is 'PeerChannel'
 *   - An array of UserInfo objects if type is 'PeerUser'
 *
 * @description
 * For channels (type='PeerChannel'):
 * - Retrieves only broadcast channels
 * - Returns channel details like title, username, ID, access hash, creator status etc.
 *
 * For users (type='PeerUser'):
 * - Retrieves regular user chats
 * - Returns user details like name, online status, last seen, unread count etc.
 * - Filters out bots and users without first names
 */
export async function getUserChats<T extends Dialog['peer']['className']>(
	client: TelegramClient,
	type: T
): Promise<T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]> {
	if (!client.connected) {
		await client.connect();
	}

	const result = (await client.invoke(
		new Api.messages.GetDialogs({
			offsetDate: 0,
			offsetId: 0,
			offsetPeer: new Api.InputPeerEmpty(),
			limit: 30000
		})
	)) as unknown as MessagesResponse;

	if (type === 'PeerChannel') {
		const channels = result.dialogs.filter((dialog) => {
			return dialog.peer.className === 'PeerChannel';
		});
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
		).filter(({ isBroadcast }) => {
			return isBroadcast;
		}) as ChannelInfo[];
		return channelsInfo as unknown as Promise<T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]>;
	}
	if (type === 'PeerUser') {
		const userChats = result.dialogs.filter((dialog) => {
			return dialog.peer.className === 'PeerUser';
		});
		const users = await Promise.all(
			userChats.map(async ({ peer, unreadCount }) => {
				try {
					const user = (await getUserInfo(client, peer.userId)) as unknown as TelegramUser | null;
					if (!user) {
						return null;
					}
					const wasOnline = user.status?.wasOnline;
					const date = wasOnline ? new Date(wasOnline * 1000) : null;

					return {
						firstName: user.firstName,
						isBot: user.bot,
						peerId: peer.userId,
						accessHash: user.accessHash as unknown as bigInt.BigInteger,
						unreadCount: unreadCount,
						lastSeen: wasOnline
							? {
								type: 'time',
								value: date!
							}
							: {
								type: 'status',
								value: user.status?.className
									? (lastSeenMessages[user.status?.className as keyof typeof lastSeenMessages] ??
										'last seen a long time ago')
									: 'last seen a long time ago'
							},
						isOnline: user.status?.className === 'UserStatusOnline'
					} satisfies UserInfo;
				} catch (err) {
					return null;
				}
			})
		);

		chatUsers = users.filter((user): user is NonNullable<typeof user> => {
			return user !== null;
		});
		return chatUsers.filter(({ isBot, firstName }) => {
			return !isBot && firstName;
		}) as unknown as Promise<T extends 'PeerChannel' ? ChannelInfo[] : UserInfo[]>;
	}
	return [];
}

export async function getUserInfo(client: TelegramClient, userId: bigInt.BigInteger) {
	try {
		if (!client.connected) {
			await client.connect();
		}
		const user = await client.getEntity(await client.getInputEntity(userId));
		return user;
	} catch (err) {
		return null;
	}
}
