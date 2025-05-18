import {
	isToday,
	isYesterday,
	isThisWeek,
	isThisYear,
	format,
	differenceInSeconds,
	differenceInCalendarDays
} from 'date-fns';
import { ChannelInfo, ChatType, FormattedMessage, UserInfo } from '../types';
import notifier from 'node-notifier';
import { getConfig } from '@/config/configManager';
import { getUserChats } from '@/telegram/client';
import { DialogInfo } from '@/telegram/client.types';
import { LRUCache } from 'lru-cache';
import { $ } from 'bun';
import os from 'node:os';

export const ICONS = {
	USER: '👤',
	CHANNEL: '📢',
	GROUP: '👥',
	MESSAGE: '💬',
	SEARCH: '🔍',
	CHECK: '✓',
	CROSS: '✗',
	ARROW: '→',
	STAR: '⭐',
	WARNING: '⚠️',
	ERROR: '❌',
	SUCCESS: '✅',
	LOADING: '⏳',
	FOLDER: '📁',
	FILE: '📄',
	LINK: '🔗',
	CLOCK: '🕐',
	HEART: '❤️',
	PIN: '📌',
	LOCK: '🔒',
	UNLOCK: '🔓'
};

export const getFilePath = async () => {
	try {
		const process = await $`zenity --file-selection --title="Select a file"`;
		const { stdout } = process;
		const stdOutStr = stdout.toString();
		return stdOutStr.trim();
	} catch (err) {
		if (err instanceof Error) {
			console.error(err.message);
		} else {
			console.error('Unknown Error');
		}
	}
};

export async function isProgramInstalled(programName: string): Promise<boolean> {
	const osName = os.platform();
	if (osName === 'win32') throw new Error('Windows is not supported yet');
	return new Promise<boolean>(async (res, rej) => {
		try {
			await $`which ${programName}`;
			res(true);
		} catch (err) {
			console.log(
				`We couldn't find ${programName} installed on your system \n Please install it and try again`
			);
			if (err instanceof Error) {
				rej(false);
			} else {
				rej(false);
			}
		}
	});
}

/**
 * Handles incoming messages and updates the chat state accordingly
 *
 * @param {Partial<FormattedMessage>} message - The incoming message with sender, content and other metadata
 * @param {Awaited<ReturnType<typeof getUserChats>>} userChats - Current state of user chats
 * @param {ChatType} currentChatType - Type of current chat (user/channel)
 * @param {React.Dispatch<React.SetStateAction<{dialogs: UserInfo[] | ChannelInfo[]; lastDialog: DialogInfo | null;} | undefined>>} setUserChats - State setter for updating chat list
 *
 * @description
 * This function:
 * 1. Sends desktop notifications for new messages if enabled
 * 2. Updates unread count and last message for the sender 3. Updates the chat list state with new message information 4. Handles both user and channel messages differently based on chat type */
export const onMessage = (
	message: Partial<FormattedMessage>,
	userChats: Awaited<ReturnType<typeof getUserChats>>,
	currentChatType: ChatType,
	user: Omit<UserInfo, 'unreadCount'> | null,
	setUserChats: React.Dispatch<
		React.SetStateAction<
			| {
					dialogs: UserInfo[] | ChannelInfo[];
					lastDialog: DialogInfo | null;
			  }
			| undefined
		>
	>
) => {
	const sender = message.sender;
	const content = message.content;
	const isFromMe = message.isFromMe;

	if (!message.isFromMe) {
		const notificationConfig = getConfig('notifications');
		if (notificationConfig.enabled) {
			notifier.notify({
				title: notificationConfig.showMessagePreview
					? `TGCli - ${sender} sent you a message!`
					: `TGCli`,
				message: notificationConfig.showMessagePreview ? content : `${sender} sent you a message!`,
				sound: true
			});
		}
	}

	const updatedUserChats = userChats?.dialogs?.map((u) => {
		if (currentChatType === 'user') {
			const userToUpdate = u as UserInfo;
			if (userToUpdate.firstName === user?.firstName) {
				return {
					...userToUpdate,
					unreadCount: userToUpdate.unreadCount + 1,
					lastMessage: content,
					isFromMe
				};
			}
			return u;
		} else {
			const userToUpdate = u as ChannelInfo;
			if (userToUpdate.title === user?.firstName) {
				return {
					...userToUpdate,
					unreadCount: userToUpdate.unreadCount + 1,
					lastMessage: content,
					isFromMe
				};
			}
			return u;
		}
	});

	if (currentChatType === 'user') {
		setUserChats((prev) => {
			return {
				dialogs: updatedUserChats as UserInfo[] | ChannelInfo[],
				lastDialog: prev?.lastDialog ?? null
			};
		});
	}
};

type OnUserOnlineStatusParams = {
	user: {
		accessHash: string;
		firstName: string;
		status: 'online' | 'offline';
		lastSeen?: number;
	};
	currentChatType: ChatType;
	selectedUser: UserInfo | ChannelInfo | null;
	setSelectedUser: (selectedUser: UserInfo | ChannelInfo | null) => void;
	setUserChats: React.Dispatch<
		React.SetStateAction<
			| {
					dialogs: UserInfo[] | ChannelInfo[];
					lastDialog: DialogInfo | null;
			  }
			| undefined
		>
	>;
};

export const onUserOnlineStatus = (params: OnUserOnlineStatusParams) => {
	const { currentChatType, selectedUser, setSelectedUser, setUserChats, user } = params;
	const { firstName, status, lastSeen } = user;
	if (
		currentChatType === 'user' &&
		selectedUser &&
		'firstName' in selectedUser &&
		firstName === selectedUser.firstName
	) {
		const date = lastSeen ? new Date(lastSeen * 1000) : null;
		const user = {
			...selectedUser,
			isOnline: status === 'online',
			lastSeen: date ? { type: 'time', value: date } : null
		} satisfies UserInfo;
		setSelectedUser(user);
	}

	setUserChats((prv) => {
		const dialog = prv?.dialogs as UserInfo[];
		const updatedData = dialog?.map((u) => {
			if (u.firstName === firstName) {
				const date = lastSeen ? new Date(lastSeen * 1000) : null;
				const user = {
					...u,
					isOnline: status === 'online',
					lastSeen: date ? { type: 'time', value: date } : null
				} satisfies UserInfo;
				return user;
			}
			return u;
		});
		return {
			dialogs: updatedData,
			lastDialog: prv?.lastDialog ?? null
		};
	});
};

export const cache = new LRUCache<string, DialogInfo[]>({
	max: 1000,
	ttl: 1000 * 60 * 5
});

export function formatLastSeen(lastSeen: UserInfo['lastSeen']) {
	if (lastSeen?.type === 'status') {
		return lastSeen.value;
	}
	const date = lastSeen?.value;

	if (!(date instanceof Date)) {
		return 'Invalid date';
	}

	const secondsDiff = differenceInSeconds(new Date(), date);
	if (secondsDiff < 60) {
		return 'last seen just now';
	}

	if (isToday(date)) {
		return `last seen at ${format(date, 'h:mm a')}`;
	}

	if (isYesterday(date)) {
		return `last seen yesterday at ${format(date, 'h:mm a')}`;
	}

	const daysDiff = differenceInCalendarDays(new Date(), date);
	if (daysDiff < 7 && isThisWeek(date)) {
		return `last seen on ${format(date, 'EEEE')} at ${format(date, 'h:mm a')}`;
	}
	if (isThisYear(date)) {
		return `last seen on ${format(date, 'MMM d')} at ${format(date, 'h:mm a')}`;
	}
	return `last seen on ${format(date, 'MMM d, yyyy')} at ${format(date, 'h:mm a')}`;
}
