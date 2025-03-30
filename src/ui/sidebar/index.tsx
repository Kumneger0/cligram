import { getConfig } from '@/config/configManager';
import { useTGCliStore } from '@/lib/store';
import { ChannelInfo, FormattedMessage, UserInfo } from '@/lib/types';
import { ICONS } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import { getUserChats } from '@/telegram/client';
import { listenForEvents } from '@/telegram/messages';
import { Box, Text, useFocus, useInput } from 'ink';
import notifier from 'node-notifier';
import React, { useCallback, useEffect, useState } from 'react';


// This is the number of empty spaces at the bottom of the sidebar and the top of the sidebar 🔍
//TODO: I KNOW this name sucks, i'll change it later if you have any suggestions please make a pr 🙏
const HEIGHT_EMPTY_SPACE = 10

export function Sidebar({ height }: { height: number; width: number }) {
	const client = useTGCliStore((state) => {
		return state.client;
	})!;
	const setSelectedUser = useTGCliStore((state) => {
		return state.setSelectedUser;
	});
	const selectedUser = useTGCliStore((state) => {
		return state.selectedUser;
	});

	const [activeChat, setActiveChat] = useState<UserInfo | ChannelInfo | null>(null);

	const [offset, setOffset] = useState(0);

	const { isFocused } = useFocus({ id: componenetFocusIds.sidebar });

	const setSearchMode = useTGCliStore((state) => {
		return state.setSearchMode;
	});
	const [userChats, setUserChats] = useState<Awaited<ReturnType<typeof getUserChats>>>()

	const setCurrentlyFocused = useTGCliStore((state) => {
		return state.setCurrentlyFocused;
	});

	const currentChatType = useTGCliStore((state) => {
		return state.currentChatType;
	});


	const setCurrentChatType = useTGCliStore((state) => {
		return state.setCurrentChatType;
	});

	const onMessage = useCallback((message: Partial<FormattedMessage>) => {
		const sender = message.sender;
		const content = message.content;
		const isFromMe = message.isFromMe;

		if (!message.isFromMe) {
			const notificationConfig = getConfig('notifications');
			if (notificationConfig.enabled) {
				notifier.notify({
					title: notificationConfig.showMessagePreview ? `TGCli - ${sender} sent you a message!` : `TGCli`,
					message: notificationConfig.showMessagePreview ? content : `${sender} sent you a message!`,
					sound: true
				});
			}
		}

		const updatedUserChats = userChats?.dialogs?.map((user) => {
			if (currentChatType === 'user') {
				const userToUpdate = user as UserInfo
				if (userToUpdate.firstName === sender) {
					return {
						...userToUpdate,
						unreadCount: userToUpdate.unreadCount + 1,
						lastMessage: content,
						isFromMe
					};
				}
				return user;
			} else {
				const userToUpdate = user as ChannelInfo
				if (userToUpdate.title === sender) {
					return {
						...userToUpdate,
						unreadCount: userToUpdate.unreadCount + 1,
						lastMessage: content,
						isFromMe
					};
				}
				return user;
			}
		});


		if (currentChatType === 'user') {
			setUserChats((prev) => {
				return {
					dialogs: updatedUserChats as UserInfo[] | ChannelInfo[],
					lastDialog: prev?.lastDialog ?? null
				}
			})
		}
	}, []);

	useEffect(() => {
		if (isFocused) {
			setCurrentlyFocused('sidebar');
		}
	}, [isFocused]);

	const onUserOnlineStatus = ({
		status,
		lastSeen,
		firstName
	}: {
		accessHash: string;
		firstName: string;
		status: 'online' | 'offline';
		lastSeen?: number;
		}) => {
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
			const dialog = (prv?.dialogs) as UserInfo[]
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
			}
		});
	};

	useEffect(() => {
		let unsubscribe: (() => void) | undefined;
		const getChats = async () => {
			const result = await getUserChats(client, currentChatType)
			setUserChats(result)
			setActiveChat(result.dialogs?.[0] ?? null)
			if (!selectedUser) {
				setSelectedUser(result.dialogs?.[0] ?? null)
			}
		};
		getChats().then(async () => {
			if (currentChatType === 'user') {
				unsubscribe = await listenForEvents(client, { onMessage, onUserOnlineStatus });
			}
		});
		return () => {
			return unsubscribe?.();
		};
	}, []);

	useInput(async (input, key) => {
		if (!isFocused) {
			return;
		}
		if (key.ctrl && input === 'k') {
			setSearchMode('CHANNELS_OR_ USERS');
		}

		if (input === 'c') {
			setCurrentChatType('channel');
		}
		if (input === 'u') {
			setCurrentChatType('user');
		}

		if (input === 'g') {
			setCurrentChatType('group');
		}

		if (key.return) {
			setSelectedUser(activeChat);
			setOffset(0);
		}

		if (key.upArrow || input === 'k') {
			if (currentChatType === 'user') {
				const currentIndex = (userChats?.dialogs as UserInfo[]).findIndex(({ peerId }) => {
					return peerId === (activeChat as UserInfo).peerId;
				});

				const nextUser = (userChats?.dialogs as UserInfo[])[currentIndex - 1];
				if (nextUser) {
					setOffset((prev) => {
						return Math.max(prev - 1, 0);
					});
					setActiveChat(nextUser);
				}
			} else {
				const currentSelectedId = (activeChat as ChannelInfo).channelId;
				const currentIndex = (userChats?.dialogs as ChannelInfo[]).findIndex(({ channelId }) => {
					return channelId === currentSelectedId;
				});
				const nextChannel = (userChats?.dialogs as ChannelInfo[])[currentIndex - 1];
				if (nextChannel) {
					setOffset((prev) => {
						return Math.max(prev - 1, 0);
					});
					setActiveChat(nextChannel);
				}
			}
		}

		if (key.downArrow || input === 'j') {
			if (currentChatType === 'user') {
				const currentIndex = (userChats?.dialogs as UserInfo[]).findIndex(({ peerId }) => {
					return peerId === (activeChat as UserInfo).peerId;
				});
				const nextUser = (userChats?.dialogs as UserInfo[])[currentIndex + 1];
				const leastAmountOfChatsTobeDisplayed = height - HEIGHT_EMPTY_SPACE
				const visibleChats = (userChats?.dialogs as UserInfo[])?.slice(offset)
				const shouldWeIncrementOffset = visibleChats.length > leastAmountOfChatsTobeDisplayed

				if (shouldWeIncrementOffset) {
					setOffset((prev) => {
						return prev + 1;
					});
				}
				if (nextUser) {
					setActiveChat(nextUser);
				}
			} else {
				const currentSelectedId = (activeChat as ChannelInfo).channelId;
				const currentIndex = (userChats?.dialogs as ChannelInfo[]).findIndex(({ channelId }) => {
					return channelId === currentSelectedId;
				});
				const nextChannel = (userChats?.dialogs as ChannelInfo[])[currentIndex + 1];

				if (
					currentIndex + 1 > height &&
					(userChats?.dialogs as ChannelInfo[]).length > height &&
					currentIndex + 1 < (userChats?.dialogs as ChannelInfo[]).length
				) {
					setOffset((prev) => {
						return prev + 1;
					});
				}
				if (nextChannel) {
					setActiveChat(nextChannel);
				}
			}
		}
	});

	const visibleChats =
		(currentChatType === 'user'
			? (userChats?.dialogs as UserInfo[])?.slice(offset, offset + height)
			: (userChats?.dialogs as ChannelInfo[])?.slice(offset, offset + height)) ?? []

	return (
		<Box
			width={height}
			flexDirection="column"
			height={height}
			borderStyle={isFocused ? 'round' : undefined}
			borderColor={isFocused ? 'green' : ''}
		>
			<Text color="blue" bold>
				{currentChatType === 'user' ? 'Chats' : currentChatType === 'group' ? 'Groups' : 'Channels'}
			</Text>
			{visibleChats.map((chat) => {
				const isChannel = currentChatType === 'channel';
				const isGroup = currentChatType === 'group';

				const id = isChannel || isGroup ? (chat as ChannelInfo).channelId : (chat as UserInfo).peerId;
				const isOnline = currentChatType === 'user' ? (chat as UserInfo).isOnline : false;
				const name = isChannel || isGroup ? (chat as ChannelInfo).title : (chat as UserInfo).firstName;
				const isSelected = isChannel || isGroup
					? (activeChat as ChannelInfo | null)?.channelId === id
					: (activeChat as UserInfo | null)?.peerId === id;

				return (
					<Box overflowY="hidden" key={String(id)} flexDirection="column">
						<Text color={isSelected ? 'green' : isOnline ? 'yellow' : 'white'}>
							{isChannel ? ICONS.CHANNEL : isGroup ? ICONS.GROUP : ICONS.USER} {name}{' '}
							{chat.unreadCount > 0 && <Text color="red">({chat.unreadCount})</Text>}
						</Text>
					</Box>
				);
			})}
		</Box>
	);
}
