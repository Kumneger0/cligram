import { useTGCliStore } from '@/lib/store';
import { ICONS } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import { ChannelInfo, getUserChats } from '@/telegram/client';
import { listenForEvents } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, Text, useFocus, useInput } from 'ink';
import notifier from 'node-notifier';
import React, { useCallback, useEffect, useState } from 'react';

export function Sidebar({ height }: { height: number; width: number }) {
	const client = useTGCliStore((state) => state.client)!;
	const setSelectedUser = useTGCliStore((state) => state.setSelectedUser);
	const selectedUser = useTGCliStore((state) => state.selectedUser);

	const [activeChat, setActiveChat] = useState<ChatUser | ChannelInfo | null>(null);
	const [chatUsers, setChatUsers] = useState<(ChatUser & { unreadCount: number })[]>([]);
	const [offset, setOffset] = useState(0);
	const { isFocused } = useFocus({ id: componenetFocusIds.sidebar });
	const setSearchMode = useTGCliStore((state) => state.setSearchMode);

	const [channels, setChannels] = useState<ChannelInfo[]>([]);

	const currentChatType = useTGCliStore((state) => state.currentChatType);
	const setCurrentChatType = useTGCliStore((state) => state.setCurrentChatType);

	const onMessage = useCallback((message: Partial<FormattedMessage>) => {
		const sender = message.sender;
		const content = message.content;
		const isFromMe = message.isFromMe;

		if (!message.isFromMe) {
			notifier.notify({
				title: `TGCli - ${sender} sent you a message!`,
				message: content,
				sound: true
			});
		}
		setChatUsers((prev) =>
			prev.map((user) => {
				if (user.firstName === sender) {
					return {
						...user,
						unreadCount: user.unreadCount + 1,
						lastMessage: content,
						isFromMe
					};
				}
				return user;
			})
		);
	}, []);

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
			firstName == selectedUser?.firstName
		) {
			const date = lastSeen ? new Date(lastSeen * 1000) : null;
			const user = { ...selectedUser, isOnline: status == 'online', lastSeen: date };
			setSelectedUser(user);
		}
		setChatUsers((prv) => {
			const updatedData = prv.map((u) => {
				if (u.firstName == firstName) {
					const date = lastSeen ? new Date(lastSeen * 1000) : null;
					const user = { ...u, isOnline: status == 'online', lastSeen: date };
					return user;
				}
				return u;
			});
			return updatedData;
		});
	};

	useEffect(() => {
		let unsubscribe: () => void;
		const getChats = async () => {
			const users = await getUserChats(client, currentChatType);
			if (currentChatType === 'PeerChannel') {
				setChannels(users as ChannelInfo[]);
				setActiveChat(users[0] as ChannelInfo);
			} else {
				setChatUsers(users as ChatUser[]);
				setActiveChat(users[0] as ChatUser);
			}
		};
		getChats().then(async () => {
			if (currentChatType === 'PeerUser') {
				unsubscribe = await listenForEvents(client, { onMessage, onUserOnlineStatus });
			}
		});
		return () => unsubscribe?.();
	}, []);

	useInput((input, key) => {
		if (!isFocused) return;

		if (input === 'c') {
			setCurrentChatType('PeerChannel');
		}
		if (input === 'u') {
			setCurrentChatType('PeerUser');
		}

		if (input === 'f') {
			setSearchMode('CHANNELS_OR_ USERS');
		}

		if (key.return) {
			setSelectedUser(activeChat);
			setOffset(0);
		}

		if (key.upArrow || input === 'k') {
			if (currentChatType === 'PeerUser') {
				const currentIndex = chatUsers.findIndex(
					({ peerId }) => peerId === (activeChat as ChatUser)?.peerId
				);

				console.log('currentIndex', currentIndex);

				const nextUser = chatUsers[currentIndex - 1];
				if (nextUser) {
					setOffset((prev) => Math.max(prev - 1, 0));
					setActiveChat(nextUser);
				}
			} else {
				const currentSelectedId = (activeChat as ChannelInfo)?.channelId;
				const currentIndex = channels.findIndex(({ channelId }) => channelId === currentSelectedId);
				const nextChannel = channels[currentIndex - 1];
				if (nextChannel) {
					setOffset((prev) => Math.max(prev - 1, 0));
					setActiveChat(nextChannel);
				}
			}
		}

		if (key.downArrow || input === 'j') {
			if (currentChatType === 'PeerUser') {
				const currentIndex = chatUsers.findIndex(
					({ peerId }) => peerId === (activeChat as ChatUser)?.peerId
				);

				console.log('currentIndex', currentIndex);
				const nextUser = chatUsers[currentIndex + 1];

				if (
					currentIndex + 1 > height &&
					chatUsers.length > height &&
					currentIndex + 1 < chatUsers.length
				) {
					setOffset((prev) => prev + 1);
				}
				if (nextUser) {
					setActiveChat(nextUser);
				}
			} else {
				const currentSelectedId = (activeChat as ChannelInfo)?.channelId;
				const currentIndex = channels.findIndex(({ channelId }) => channelId === currentSelectedId);
				const nextChannel = channels[currentIndex + 1];

				if (
					currentIndex + 1 > height &&
					channels.length > height &&
					currentIndex + 1 < channels.length
				) {
					setOffset((prev) => prev + 1);
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

				const id = isChannel ? (chat as ChannelInfo).channelId : (chat as ChatUser).peerId;
				const isOnline = currentChatType === 'PeerUser' ? (chat as ChatUser).isOnline : false;
				const name = isChannel ? (chat as ChannelInfo).title : (chat as ChatUser).firstName;
				const isSelected =
					isChannel
						? (activeChat as ChannelInfo)?.channelId == id
						: (activeChat as ChatUser)?.peerId == id;

				return (
					<Box overflowY="hidden" key={String(id)} flexDirection="column">
						<Text color={isSelected ? 'green' : isOnline ? 'yellow' : 'white'}>
							{isChannel ? ICONS.CHANNEL : ICONS.USER} {name} {chat.unreadCount > 0 && <Text color="red">({chat.unreadCount})</Text>}
						</Text>
					</Box>
				);
			})}
		</Box>
	);
}
