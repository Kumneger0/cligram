import { useTGCliStore } from '@/lib/store';
import { ChannelInfo, UserInfo } from '@/lib/types';
import { ICONS } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import { getUserChats } from '@/telegram/client';
import { Box, Text, useFocus, useInput } from 'ink';
import React, { useEffect, useState } from 'react';

// This is the number of empty spaces at the bottom of the sidebar and the top of the sidebar 🔍
//TODO: I KNOW this name sucks, i'll change it later if you have any suggestions please make a pr 🙏
const HEIGHT_EMPTY_SPACE = 10;

export function Sidebar({ height, userChats }: { height: number; width: number, userChats: Awaited<ReturnType<typeof getUserChats>> | undefined }) {
	const setSelectedUser = useTGCliStore((state) => {
		return state.setSelectedUser;
	});

	const [activeChat, setActiveChat] = useState<UserInfo | ChannelInfo | null>(() => {
		return userChats?.dialogs?.[0] ?? null;
	});

	const [offset, setOffset] = useState(0);

	const { isFocused } = useFocus({ id: componenetFocusIds.sidebar });

	const setSearchMode = useTGCliStore((state) => {
		return state.setSearchMode;
	});

	const setCurrentlyFocused = useTGCliStore((state) => {
		return state.setCurrentlyFocused;
	});

	const currentChatType = useTGCliStore((state) => {
		return state.currentChatType;
	});

	const setCurrentChatType = useTGCliStore((state) => {
		return state.setCurrentChatType;
	});


	useEffect(() => {
		if (isFocused) {
			setCurrentlyFocused('sidebar');
		}
	}, [isFocused]);


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
				const leastAmountOfChatsTobeDisplayed = height - HEIGHT_EMPTY_SPACE;
				const visibleChats = (userChats?.dialogs as UserInfo[])?.slice(offset);
				const shouldWeIncrementOffset = visibleChats.length > leastAmountOfChatsTobeDisplayed;

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
			: (userChats?.dialogs as ChannelInfo[])?.slice(offset, offset + height)) ?? [];

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
			{visibleChats.map((chat, index) => {
				const isChannel = currentChatType === 'channel';
				const isGroup = currentChatType === 'group';

				const id =
					isChannel || isGroup ? (chat as ChannelInfo).channelId : (chat as UserInfo).peerId;
				const isOnline = currentChatType === 'user' ? (chat as UserInfo).isOnline : false;
				const name =
					isChannel || isGroup ? (chat as ChannelInfo).title : (chat as UserInfo).firstName;
				const isSelected =
					isChannel || isGroup
						? (activeChat as ChannelInfo | null)?.channelId === id
						: (activeChat as UserInfo | null)?.peerId === id;

				return (
					<Box overflowY="hidden" key={index} flexDirection="column">
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
