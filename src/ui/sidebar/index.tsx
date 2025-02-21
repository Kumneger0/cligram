import { useTGCliStore } from '@/lib/store';
import { componenetFocusIds } from '@/lib/utils/consts';
import { getUserChats } from '@/telegram/client';
import { listenForEvents } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, Text, useFocus, useInput } from 'ink';
import notifier from 'node-notifier';
import React, { useCallback, useEffect, useState } from 'react';

export function Sidebar({ height, width }: { height: number, width: number }) {
	const client = useTGCliStore((state) => state.client)!;
	const setSelectedUser = useTGCliStore((state) => state.setSelectedUser);
	const selectedUser = useTGCliStore((state) => state.selectedUser)

	const [activeChat, setActiveChat] = useState<ChatUser | null>(null);
	const [chatUsers, setChatUsers] = useState<(ChatUser & { unreadCount: number })[]>([]);
	const [offset, setOffset] = useState(0);

	const { isFocused } = useFocus({ id: componenetFocusIds.sidebar });

	const onMessage = useCallback((message: Partial<FormattedMessage>) => {
		const sender = message.sender;
		const content = message.content;
		const isFromMe = message.isFromMe;

		if (!message.isFromMe) {
			notifier.notify({
				title: `TGCli - ${sender} sent you a message!`,
				message: content,
				sound: true,
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


	const onUserOnlineStatus = ({ status, lastSeen, firstName }: { accessHash: string; firstName: string; status: "online" | "offline"; lastSeen?: number }) => {
		if (firstName == selectedUser?.firstName) {
			const date = lastSeen ? new Date(lastSeen * 1000) : null
			const user = { ...selectedUser, isOnline: status == "online", lastSeen: date }
			setSelectedUser(user)
		}
		setChatUsers((prv) => {
			const updatedData = prv.map((u) => {
				if (u.firstName == firstName) {
					const date = lastSeen ? new Date(lastSeen * 1000) : null
					const user = { ...u, isOnline: status == "online", lastSeen: date }
					return user
				}
				return u
			})
			return updatedData
		})
	}

	useEffect(() => {
		let unsubscribe: () => void
		const getChats = async () => {
			const users = (await getUserChats(client)).filter(
				({ isBot, firstName }) => !isBot && firstName
			);
			setChatUsers(users);
			if (users.length > 0) {
				setSelectedUser(users[0]!);
			}
		};
		getChats().then(async () => {
			unsubscribe = await listenForEvents(client, { onMessage, onUserOnlineStatus });
		});
		return () => unsubscribe()
	}, []);

	useInput((input, key) => {
		if (!isFocused) return;
		if (key.return) {
			setSelectedUser(activeChat);
		}
		if (key.upArrow || input === 'k') {
			const currentIndex = chatUsers.findIndex(({ peerId }) => peerId === activeChat?.peerId);
			const nextUser = chatUsers[currentIndex - 1];
			if (nextUser) {
				setOffset((prev) => Math.max(prev - 1, 0));
				setActiveChat(nextUser);
			}
		} else if (key.downArrow || input === 'j') {
			const currentIndex = chatUsers.findIndex(({ peerId }) => peerId === activeChat?.peerId);
			const nextUser = chatUsers[currentIndex + 1];

			if (currentIndex + 1 > height && chatUsers.length > height && currentIndex + 1 < chatUsers.length) {
				setOffset((prev) => prev + 1);
			}

			if (nextUser) {
				setActiveChat(nextUser);
			}
		}
	});
	const visibleChats = chatUsers.slice(offset, offset + height);

	return (
		<Box
			width={height}
			flexDirection="column"
			height={height}
			borderStyle={isFocused ? 'round' : undefined}
			borderColor={isFocused ? 'green' : ''}
		>
			<Text color="blue" bold>
				Chats
			</Text>
			{visibleChats.map(({ firstName, peerId, unreadCount, isOnline }) => {
				return <Box overflowY='hidden' key={String(peerId)} flexDirection="column">
					<Text color={activeChat?.peerId == peerId ? 'green' : isOnline ? "yellow" : "white"} >
						{activeChat?.peerId == peerId && isFocused ? '>' : null} {firstName}{' '}
						{unreadCount > 0 && <Text color="red">({unreadCount})</Text>}

					</Text>
				</Box>
			})}
		</Box>
	);
}
