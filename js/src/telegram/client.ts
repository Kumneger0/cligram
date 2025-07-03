import { getConfig } from '@/config/configManager.js';
import { cache, formatLastSeen } from '@/lib/utils/index.js';
import { Api, TelegramClient } from 'telegram';
import { Channel, ChannelInfo, ChatType, TelegramUser, UserInfo } from '../lib/types/index.js';
import { DialogInfo } from './client.types.js';
import bigInt from 'big-integer';
import { LRUCache } from 'lru-cache';

export let chatUsers: UserInfo[] = [];

//we are fetching diloags at once so no need to cache users, channels, groups separately we can cache all of them together
const CACHE_KEY = 'dialogs';

export const lastSeenMessages = {
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
				offline: !online
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

export async function getChannelInfo(client: TelegramClient, channelId: bigInt.BigInteger) {
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
export async function searchUsers(
	client: TelegramClient,
	query: string
): Promise<{ users: UserInfo[], channels: ChannelInfo[] }> {
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
				peerId: user.id.toString(),
				accessHash: user?.accessHash?.toString() ?? "",
				isBot: user.bot ?? false,
				unreadCount: 0,
				lastSeen: "",
				isOnline: false
			} satisfies UserInfo;
		});
	//TODO: i'll come back later and fix this
	// const chats = result.chats
	// 	.filter((chat) => {
	// 		return chat.className === 'Chat';
	// 	})
	// 	.map((chat) => {
	// 		return {
	// 			title: chat.title || '',
	// 			chatId: chat.id as bigInt.BigInteger
	// 		};
	// 	});

	//TODO: i'll come back later and fix this
	//@ts-ignore 
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

	return { users, channels }
}

/**
 * Retrieves user chats or channels from Telegram based on the specified type.
 *
 * @template T - The type of peer to retrieve ('channel' or 'user')
 * @param {TelegramClient} client - The Telegram client instance
 * @param {T} type - The type of peers to retrieve ('channel' or 'user')
 * @returns {Promise< T extends 'channel' ? ChannelInfo[] : UserInfo[]>} A promise that resolves to:
 *   - dialogs: Array of channel info or user info objects
 */
/**
 * Retrieves a list of chats for the connected Telegram client, filtered by the specified chat type.
 *
 * - If `type` is `'channel'` or `'group'`, returns an array of `ChannelInfo` objects.
 * - If `type` is `'user'`, returns an array of `UserInfo` objects.
 *
 * The function uses a cache to avoid unnecessary network requests and fetches additional information
 * for each chat as needed.
 *
 * @template T - The chat type to filter by ('channel', 'group', or 'user').
 * @param client - The Telegram client instance. Will connect if not already connected.
 * @param type - The type of chats to retrieve: 'channel', 'group', or 'user'.
 * @returns A promise that resolves to an array of `ChannelInfo` or `UserInfo` depending on the `type` parameter.
 */
export async function getUserChats<T extends ChatType>(
	client: TelegramClient,
	type: T
): Promise<T extends 'channel' | 'group' ? ChannelInfo[] : UserInfo[]> {
	if (!client.connected) {
		await client.connect();
	}
	const cached = cache.get(CACHE_KEY);
	const result = cached ?? ((await client.getDialogs({})) as unknown as DialogInfo[]);
	cache.set(CACHE_KEY, result);
	if (type === 'channel' || type === 'group') {
		const groupOrChannels =
			type === 'channel'
				? result.filter((dialog) => {
					return dialog.dialog.peer.className === 'PeerChannel';
				})
				: result.filter((dialog) => {
					return dialog.isGroup;
				});

		const channelsInfo = await Promise.all(
			groupOrChannels.map(async (chan) => {
				const id =
					'channelId' in chan.dialog.peer
						? (chan.dialog.peer as { channelId: bigInt.BigInteger }).channelId
						: (chan.dialog.peer as { chatId: bigInt.BigInteger }).chatId;
				const isPeerChat = chan.dialog.peer.className === 'PeerChat';
				const channel = !isPeerChat
					? ((await getChannelInfo(client, id)) as unknown as Channel)
					: null;
				if (isPeerChat) {
					return {
						title: chan.title,
						username: '',
						channelId: (chan.dialog.peer as { chatId: bigInt.BigInteger }).chatId.toString(),
						accessHash: '',
						isCreator: channel?.creator ?? false,
						isBroadcast: false,
						participantsCount:
							(chan.entity as unknown as { participantsCount: number }).participantsCount ?? 0,
						unreadCount: chan.unreadCount
					} satisfies ChannelInfo;
				}
				return {
					title: channel?.title ?? '',
					username: channel?.username ?? '',
					channelId: channel?.id.toString() ?? '',
					accessHash: channel?.accessHash.toString() ?? '',
					isCreator: channel?.creator ?? false,
					isBroadcast: channel?.broadcast ?? false,
					participantsCount:
						(chan.entity as unknown as { participantsCount: number }).participantsCount ?? 0,
					unreadCount: chan.unreadCount
				} satisfies ChannelInfo;
			})
		);
		return channelsInfo as T extends 'channel' | 'group' ? ChannelInfo[] : UserInfo[]

	}

	if (type === 'user') {
		const userChats = result.filter((dialog) => {
			return dialog.dialog.peer.className === 'PeerUser';
		});
		const users = await Promise.all(
			userChats.map(async ({ dialog, unreadCount }) => {
				try {
					const user = await getUserInfo(
						client,
						(dialog.peer as { userId: bigInt.BigInteger }).userId
					);
					if (!user) {
						return null;
					}
					return {
						...user,
						accessHash: user.accessHash.toString(),
						unreadCount: unreadCount
					} satisfies UserInfo;
				} catch (err) {
					return null;
				}
			})
		);

		chatUsers = users
			.filter((user): user is NonNullable<typeof user> => {
				return user !== null;
			})
			.filter(({ isBot, firstName }) => {
				return !isBot && firstName;
			});

		return chatUsers as T extends 'channel' | 'group' ? ChannelInfo[] : UserInfo[]

	}
	return [] as unknown as T extends 'channel' | 'group' ? ChannelInfo[] : UserInfo[]

}

const userInfoCache = new LRUCache<string, UserInfo>({
	max: 100,
	ttl: 1000 * 60 * 5
});
export async function getUserInfo(
	client: TelegramClient,
	userId: bigInt.BigInteger
): Promise<UserInfo | null> {
	const userIdString = userId.toString();
	if (userInfoCache.has(userIdString)) {
		return userInfoCache.get(userIdString)!;
	}
	try {
		if (!client.connected) {
			await client.connect();
		}
		const user = (await client.getEntity(
			await client.getInputEntity(userId)
		)) as unknown as TelegramUser | null;
		if (!user) {
			return null;
		}
		const wasOnline = user.status?.wasOnline;
		const date = wasOnline ? new Date(wasOnline * 1000) : null;

		const lastSeen = wasOnline
			? {
				type: 'time' as const,
				value: date!
			}
			: {
				type: 'status' as const,
				value: user.status?.className
					? (lastSeenMessages[user.status?.className as keyof typeof lastSeenMessages] ??
						'last seen a long time ago')
					: 'last seen a long time ago'
			}


		const userInfo = {
			firstName: user.firstName,
			isBot: user.bot,
			peerId: userId.toString() ?? "",
			accessHash: user?.accessHash?.toString(),
			lastSeen: formatLastSeen(lastSeen),
			isOnline: user.status?.className === 'UserStatusOnline',
			unreadCount: 0
		} satisfies UserInfo;
		userInfoCache.set(userIdString, userInfo);
		return userInfo;
	} catch (err) {
		return null;
	}
}
