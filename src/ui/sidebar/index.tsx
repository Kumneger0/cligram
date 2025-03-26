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
	const [chatUsers, setChatUsers] = useState<(UserInfo & { unreadCount: number })[]>([]);
	const [offset, setOffset] = useState(0);
	const { isFocused } = useFocus({ id: componenetFocusIds.sidebar });
	const setSearchMode = useTGCliStore((state) => {
		return state.setSearchMode;
	});

	const setCurrentlyFocused = useTGCliStore((state) => {
		return state.setCurrentlyFocused;
	});

	const [channels, setChannels] = useState<ChannelInfo[]>([]);

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
		setChatUsers((prev) => {
			return prev.map((user) => {
				if (user.firstName === sender) {
					return {
						...user,
						unreadCount: user.unreadCount + 1,
						lastMessage: content,
						isFromMe
					};
				}
				return user;
			});
		});
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
			currentChatType === 'PeerUser' &&
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
		setChatUsers((prv) => {
			const updatedData = prv.map((u) => {
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
			return updatedData;
		});
	};

	useEffect(() => {
		let unsubscribe: (() => void) | undefined;
		const getChats = async () => {
			const users = await getUserChats(client, currentChatType);
			if (currentChatType === 'PeerChannel') {
				setChannels(users as ChannelInfo[]);
				setActiveChat(users[0] as ChannelInfo);
			} else {
				setChatUsers(users as UserInfo[]);
				setActiveChat(users[0] as UserInfo);
			}
		};
		getChats().then(async () => {
			if (currentChatType === 'PeerUser') {
				unsubscribe = await listenForEvents(client, { onMessage, onUserOnlineStatus });
			}
		});
		return () => {
			return unsubscribe?.();
		};
	}, []);

	useInput((input, key) => {
		if (!isFocused) {
			return;
		}

		if (key.ctrl && input === 'k') {
			setSearchMode('CHANNELS_OR_ USERS');
		}

		if (input === 'c') {
			setCurrentChatType('PeerChannel');
		}
		if (input === 'u') {
			setCurrentChatType('PeerUser');
		}

		if (key.return) {
			setSelectedUser(activeChat);
			setOffset(0);
		}

		if (key.upArrow || input === 'k') {
			if (currentChatType === 'PeerUser') {
				const currentIndex = chatUsers.findIndex(({ peerId }) => {
					return peerId === (activeChat as UserInfo).peerId;
				});

				const nextUser = chatUsers[currentIndex - 1];
				if (nextUser) {
					setOffset((prev) => {
						return Math.max(prev - 1, 0);
					});
					setActiveChat(nextUser);
				}
			} else {
				const currentSelectedId = (activeChat as ChannelInfo).channelId;
				const currentIndex = channels.findIndex(({ channelId }) => {
					return channelId === currentSelectedId;
				});
				const nextChannel = channels[currentIndex - 1];
				if (nextChannel) {
					setOffset((prev) => {
						return Math.max(prev - 1, 0);
					});
					setActiveChat(nextChannel);
				}
			}
		}

		if (key.downArrow || input === 'j') {
			if (currentChatType === 'PeerUser') {
				const currentIndex = chatUsers.findIndex(({ peerId }) => {
					return peerId === (activeChat as UserInfo).peerId;
				});

				const nextUser = chatUsers[currentIndex + 1];

				if (
					currentIndex + 1 > height &&
					chatUsers.length > height &&
					currentIndex + 1 < chatUsers.length
				) {
					setOffset((prev) => {
						return prev + 1;
					});
				}
				if (nextUser) {
					setActiveChat(nextUser);
				}
			} else {
				const currentSelectedId = (activeChat as ChannelInfo).channelId;
				const currentIndex = channels.findIndex(({ channelId }) => {
					return channelId === currentSelectedId;
				});
				const nextChannel = channels[currentIndex + 1];

				if (
					currentIndex + 1 > height &&
					channels.length > height &&
					currentIndex + 1 < channels.length
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
		currentChatType === 'PeerUser'
			? chatUsers.slice(offset, offset + height)
			: channels.slice(offset, offset + height);

	return (
		<Box
			width={height}
			flexDirection="column"
			height={height}
			borderStyle={isFocused ? 'round' : undefined}
			borderColor={isFocused ? 'green' : ''}
		>
			<Text color="blue" bold>
				{currentChatType === 'PeerUser' ? 'Chats' : 'Channels'}
			</Text>
			{visibleChats.map((chat) => {
				const isChannel = currentChatType === 'PeerChannel';

				const id = isChannel ? (chat as ChannelInfo).channelId : (chat as UserInfo).peerId;
				const isOnline = currentChatType === 'PeerUser' ? (chat as UserInfo).isOnline : false;
				const name = isChannel ? (chat as ChannelInfo).title : (chat as UserInfo).firstName;
				const isSelected = isChannel
					? (activeChat as ChannelInfo | null)?.channelId === id
					: (activeChat as UserInfo | null)?.peerId === id;

				return (
					<Box overflowY="hidden" key={String(id)} flexDirection="column">
						<Text color={isSelected ? 'green' : isOnline ? 'yellow' : 'white'}>
							{isChannel ? ICONS.CHANNEL : ICONS.USER} {name}{' '}
							{chat.unreadCount > 0 && <Text color="red">({chat.unreadCount})</Text>}
						</Text>
					</Box>
				);
			})}
		</Box>
	);
}
