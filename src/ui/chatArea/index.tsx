import { getConfig } from '@/config/configManager';
import { conversationStore, useForwardMessageStore, useTGCliStore } from '@/lib/store';
import { ChannelInfo, FormattedMessage, UserInfo } from '@/lib/types';
import { formatLastSeen } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import { getUserInfo } from '@/telegram/client';
import {
	editMessage,
	getAllMessages,
	listenForEvents,
	markUnRead,
	sendMessage,
	setUserTyping
} from '@/telegram/messages';
import { Box, Text, useFocus, useFocusManager, useInput } from 'ink';
import Spinner from 'ink-spinner';
import TextInput from 'ink-text-input';
import React, { Fragment, useEffect, useLayoutEffect, useState } from 'react';
import { TelegramClient } from 'telegram';
import { MessageActionModal } from '../modal/Modal';
const formatDate = (date: Date) => {
	return date.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		year: 'numeric'
	});
};

const groupMessagesByDate = (messages: FormattedMessage[]) => {
	const sortedMessages = messages.sort((a, b) => {
		return a.date.getTime() - b.date.getTime();
	});

	return sortedMessages.reduce<Record<string, FormattedMessage[]>>((groups, message) => {
		const dateKey = formatDate(message.date);
		if (!groups[dateKey]) {
			groups[dateKey] = [];
		}
		groups[dateKey].push(message);
		return groups;
	}, {});
};

export function ChatArea({ height, width }: { height: number; width: number }) {
	const { focus } = useFocusManager();
	const { isFocused } = useFocus({ id: componenetFocusIds.chatArea });
	const selectedUser = useTGCliStore((state) => {
		return state.selectedUser;
	});
	const client = useTGCliStore((state) => {
		return state.client;
	})!;
	const { conversation, setConversation, updateConversations } = conversationStore((state) => {
		return state;
	});
	const setMessageAction = useTGCliStore((state) => {
		return state.setMessageAction;
	});
	const [offsetId, setOffsetId] = useState<number | undefined>(undefined);

	const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
	const [isLoading, setIsLoading] = useState(false);
	const [isModalOpen, setIsModalOpen] = useState(false);
	const [offset, setOffset] = useState(0);

	const setForwardMessageOptions = useForwardMessageStore((state) => {
		return state.setForwardMessageOptions;
	});

	const setCurrentlyFocused = useTGCliStore((state) => {
		return state.setCurrentlyFocused;
	});


	const setCurrentChatType = useTGCliStore((state) => {
		return state.setCurrentChatType;
	});

	const setSelectedUser = useTGCliStore((state) => {
		return state.setSelectedUser;
	});


	const currentChatType = useTGCliStore((state) => {
		return state.currentChatType;
	});
	const setSearchMode = useTGCliStore((state) => {
		return state.setSearchMode;
	});
	const currentlySelectedChatId =
		currentChatType === 'user'
			? (selectedUser as UserInfo | null)?.peerId
			: (selectedUser as ChannelInfo | null)?.channelId;

	const selectedUserPeerID = String(currentlySelectedChatId);

	async function markMessageAsRead() {
		if (selectedUser) {
			const peer = {
				peerId: (currentChatType === 'channel'
					? (selectedUser as ChannelInfo).channelId
					: (selectedUser as UserInfo).peerId) as bigInt.BigInteger,
				accessHash: selectedUser.accessHash as bigInt.BigInteger
			};
			await markUnRead({ client, peer, type: currentChatType });
		}
	}

	useEffect(() => {
		if (!selectedUser) {
			return;
		}
		setIsLoading(true);
		let unsubscribe: (() => void) | undefined;

		const id =
			currentChatType === 'user'
				? (selectedUser as UserInfo).peerId
				: ((selectedUser as ChannelInfo).channelId as unknown as bigInt.BigInteger);
		const accessHash =
			currentChatType === 'user'
				? (selectedUser as UserInfo).accessHash
				: ((selectedUser as ChannelInfo).accessHash as unknown as bigInt.BigInteger);

		(async () => {
			const conversation = await getAllMessages(
				{
					client,
					peerInfo: {
						accessHash,
						peerId: id,
						userFirtNameOrChannelTitle:
							currentChatType === 'user'
								? (selectedUser as UserInfo).firstName
								: (selectedUser as ChannelInfo).title
					},
					chatAreaWidth: width
				},
				currentChatType
			);
			const chatConfig = getConfig('chat');
			if (chatConfig.readReceiptMode === 'instant') {
				await markMessageAsRead();
			}
			setConversation(conversation);
			setOffsetId(conversation[0]?.id);
			setIsLoading(false);
			setActiveMessage(conversation.at(-1) ?? null);

			if (currentChatType === 'user') {
				unsubscribe = await listenForEvents(client, {
					onMessage: (message) => {
						const from = message.sender;
						if (from === (selectedUser as UserInfo).firstName) {
							updateConversations([message]);
							setOffsetId(message.id);
							setActiveMessage(message);
						}
					}
				});
			}
		})();

		return () => {
			unsubscribe?.();
			setConversation([]);
		};
	}, [selectedUserPeerID, currentChatType]);

	const conversationAreaHieght =
		currentChatType === 'user' ||
			(currentChatType === 'channel' && (selectedUser as ChannelInfo | null)?.isCreator) ||
			currentChatType === 'group'
			? height * (70 / 100)
			: height * (90 / 100);

	const visibleMessages = conversation.slice(offset, offset + conversationAreaHieght);

	useEffect(() => {
		if (isFocused) {
			setCurrentlyFocused('chatArea');
		}
	}, [isFocused]);

	useInput(async (input, key) => {
		if (!isFocused) {
			return;
		}

		if (input === 'f') {
			const peerId = currentlySelectedChatId as unknown as bigInt.BigInteger | undefined;
			const accessHash = selectedUser?.accessHash as unknown as bigInt.BigInteger | undefined;

			if (!peerId || !accessHash) {
				return;
			}

			setForwardMessageOptions({
				fromPeer: { peerId, accessHash },
				id: [activeMessage?.id!],
				type: currentChatType
			});
			focus(componenetFocusIds.forwardMessage);
			return;
		}

		if (input === 'd') {
			setMessageAction({ action: 'delete', id: activeMessage?.id! });
			setIsModalOpen(true);
			return;
		}
		if (key.ctrl && input === 'k') {
			setSearchMode('CHANNELS_OR_ USERS');
		}

		if (input === 'e') {
			if (!activeMessage?.isFromMe) {
				return;
			}
			setMessageAction({ action: 'edit', id: activeMessage.id! });
			focus(componenetFocusIds.messageInput);
			return;
		}

		if (input === 'r') {
			setMessageAction({ action: 'reply', id: activeMessage?.id! });
			focus(componenetFocusIds.messageInput);
			return;
		}


		if (input === 'u' && activeMessage?.fromId) {
			const user = await getUserInfo(client, activeMessage.fromId)
			setCurrentChatType('user');
			setSelectedUser({
				...user,
				unreadCount: 0
			} as UserInfo);
		}

		if (key.upArrow || input === 'k') {
			const currentIndex = conversation.findIndex(({ id }) => {
				return id === activeMessage?.id;
			});
			let nextMessage = conversation[currentIndex - 1];
			if (offset === 0 && selectedUser) {
				const appendMessages = async () => {
					const peerId =
						currentChatType === 'user'
							? (selectedUser as UserInfo).peerId
							: (selectedUser as ChannelInfo).channelId;
					const accessHash =
						currentChatType === 'user'
							? (selectedUser as UserInfo).accessHash
							: (selectedUser as ChannelInfo).accessHash;
					const userFirtNameOrChannelTitle =
						currentChatType === 'user'
							? (selectedUser as UserInfo).firstName
							: (selectedUser as ChannelInfo).title;
					const newMessages = await getAllMessages(
						{
							client,
							peerInfo: {
								accessHash: accessHash as unknown as bigInt.BigInteger,
								peerId: peerId as unknown as bigInt.BigInteger,
								userFirtNameOrChannelTitle
							},
							offsetId,
							chatAreaWidth: width
						},
						currentChatType
					);
					const updatedConversation = [...newMessages, ...conversation];
					setConversation(
						updatedConversation.filter(({ id }, i) => {
							return (
								updatedConversation.findIndex((c) => {
									return c.id === id;
								}) === i
							);
						})
					);
					nextMessage = updatedConversation[currentIndex - 1];
					setOffsetId(newMessages[0]?.id);
					setOffset(newMessages.length);
				};
				await appendMessages();
				return;
			}
			if (nextMessage) {
				setActiveMessage(nextMessage);
			}
			setOffset((prev) => {
				return Math.max(prev - 1, 0);
			});
		} else if (key.downArrow || input === 'j') {
			const currentIndex = conversation.findIndex(({ id }) => {
				return id === activeMessage?.id;
			});
			const nextMessage = conversation[currentIndex + 1];
			if (nextMessage) {
				setActiveMessage(nextMessage);
			}
			const newOffset = Math.max(offset + 1, conversation.length - conversationAreaHieght);
			if (offset < conversation.length - 1) {
				setOffset(newOffset);
			}
		}
	});

	if (isLoading) {
		return (
			<Box height={40} justifyContent="center" alignItems="center" width={'100%'}>
				<Text>
					<Text color="green">
						<Spinner type="dots" />
					</Text>
					{'Loading conversations...'}
				</Text>
			</Box>
		);
	}
	const selectedUserLastSeen =
		currentChatType === 'user'
			? (selectedUser as UserInfo | null)?.lastSeen
				? formatLastSeen((selectedUser as UserInfo).lastSeen!)
				: 'Unknown'
			: '';

	const groupedMessages = groupMessagesByDate(visibleMessages);

	return (
		<>
			{isModalOpen && (
				<MessageActionModal
					onClose={() => {
						return setIsModalOpen(false);
					}}
				/>
			)}
			{!isModalOpen && (
				<Box flexDirection="column" height={height} width={width}>
					<Box gap={1}>
						<Text color="blue" bold>
							{currentChatType === 'user'
								? (selectedUser as UserInfo | null)?.firstName
								: (selectedUser as ChannelInfo | null)?.title}{' '}
							{currentChatType === 'group' || currentChatType === 'channel' && `(${(selectedUser as ChannelInfo)?.participantsCount} Members)`}
						</Text>
						<Text>
							{currentChatType === 'user'
								? (selectedUser as UserInfo | null)?.isOnline
									? 'Online'
									: `${selectedUserLastSeen}`
								: ''}
						</Text>
					</Box>
					<Box
						width={'100%'}
						height={conversationAreaHieght}
						overflowY="hidden"
						borderStyle={isFocused ? 'classic' : undefined}
						flexDirection="column"
						gap={1}
						paddingLeft={2}
					>
						{Object.entries(groupedMessages).map(([date, messages]) => {
							return (
								<Fragment key={date.toString()}>
									<Box justifyContent="center" width={'100%'}>
										<Text>{date}</Text>
									</Box>
									{messages.map((message) => {
										const date = message.date;
										return (
											<Message
												key={message.id}
												message={message}
												date={date.toString()}
												activeMessage={activeMessage}
												isFocused={isFocused}
												client={client}
											/>
										);
									})}
								</Fragment>
							);
						})}
					</Box>
					{(currentChatType === 'user' ||
						(currentChatType === 'channel' && (selectedUser as ChannelInfo | null)?.isCreator) ||
						currentChatType === 'group') && (
						<MessageInput
							onSubmit={async (message) => {
								if (selectedUser) {
									const newMessage = {
										content: message,
										media: null,
										isFromMe: true,
										id: Math.floor(Math.random() * 10000),
										sender: 'you',
										isUnsupportedMessage: false,
										date: new Date()
									} satisfies FormattedMessage;
									setConversation([...conversation, newMessage]);
									const id =
										currentChatType === 'user'
											? (selectedUser as UserInfo).peerId
											: (selectedUser as ChannelInfo).channelId;
									const accessHash =
										currentChatType === 'user'
											? (selectedUser as UserInfo).accessHash
											: (selectedUser as ChannelInfo).accessHash;

									const chatConfig = getConfig('chat');
									if (chatConfig.readReceiptMode === 'default') {
										if (currentChatType === 'user') {
											await markMessageAsRead();
										}
									}
									await sendMessage(
										client,
										{
											peerId: id as unknown as bigInt.BigInteger,
											accessHash: accessHash as unknown as bigInt.BigInteger
										},
										message,
										undefined,
										undefined,
										currentChatType
									);
								}
							}}
						/>
					)}
				</Box>
			)}
		</>
	);
}

function Message({
	message,
	date,
	activeMessage,
	isFocused,
	client
}: {
	message: FormattedMessage;
	date: string;
	activeMessage: FormattedMessage | null;
	isFocused: boolean;
	client: TelegramClient;
}) {
	const [sender, setSender] = useState<Omit<UserInfo, 'unreadCount'> | null>(null);

	useEffect(() => {
		const getSender = async () => {
			if (message.fromId) {
				const user = await getUserInfo(client, message.fromId)
				setSender(user);
			}
		};
		getSender();
	}, [message.fromId]);





	return (
		<Box
			alignSelf={message.isFromMe ? 'flex-end' : 'flex-start'}
			key={message.id}
			width={'30%'}
			height={'auto'}
			flexShrink={0}
			flexDirection="column"
			flexGrow={0}
		>
			{activeMessage?.id === message.id && isFocused ? <Text color={'green'}>{'>  '}</Text> : null}
			<Box flexDirection="column">
				<Text
					backgroundColor={activeMessage?.id === message.id && isFocused ? 'blue' : ''}
					color={activeMessage?.id === message.id && isFocused ? 'white' : ''}
				>
					{message.media && <Text wrap="end">{message.media}</Text>}
				</Text>
				<Text
					wrap="wrap"
					color={'white'}
					bold={!!message.isUnsupportedMessage}
					underline={!!message.isUnsupportedMessage}
					backgroundColor={activeMessage?.id === message.id && isFocused ? 'blue' : ''}
				>
					{message.content}
				</Text>
				{sender && !message.isFromMe && (
					<Text>
						Sent by: {sender.firstName}
					</Text>
				)}
				<Text>
					{new Date(date).toLocaleTimeString([], {
						hour: '2-digit',
						minute: '2-digit',
						hour12: true
					})}
				</Text>
			</Box>
		</Box>
	);
}

function MessageInput({ onSubmit }: { onSubmit: (message: string) => void }) {
	const [message, setMessage] = useState('');
	const selectedUser = useTGCliStore((state) => {
		return state.selectedUser;
	});
	const { isFocused } = useFocus({ id: componenetFocusIds.messageInput });

	const messageAction = useTGCliStore((state) => {
		return state.messageAction;
	});


	const client = useTGCliStore((state) => {
		return state.client;
	})!;

	const setMessageAction = useTGCliStore((state) => {
		return state.setMessageAction;
	});

	const conversation = conversationStore((state) => {
		return state.conversation;
	});

	const messageContent = conversation.find(({ id }) => {
		return id === messageAction?.id;
	})?.content;

	const isReply = messageAction?.action === 'reply';

	const currentChatType = useTGCliStore((state) => {
		return state.currentChatType;
	});
	const setCurrentlyFocused = useTGCliStore((state) => {
		return state.setCurrentlyFocused;
	});

	useEffect(() => {
		if (isFocused) {
			setCurrentlyFocused(null);
		}
	}, [isFocused]);

	useLayoutEffect(() => {
		if (isReply) {
			return;
		}
		setMessage(messageContent ?? '');
	}, [messageAction?.id]);

	const edit = () => {
		if (selectedUser) {
			const newMessage = {
				content: message,
				media: null,
				isFromMe: true,
				id: messageAction?.id ?? Math.floor(Math.random() * 10000),
				sender: 'you',
				date: new Date(),
				isUnsupportedMessage: false
			} satisfies FormattedMessage;
			const updatedConversation = conversation.map((msg) => {
				if (msg.id === messageAction?.id) {
					return newMessage;
				}
				return msg;
			});
			conversationStore.setState({ conversation: updatedConversation });
			if (currentChatType === 'user') {
				editMessage(client, selectedUser as UserInfo, messageAction?.id!, message);
			}
			setMessageAction(null);
		}
	};
	return (
		<Box borderStyle={isFocused ? 'classic' : undefined} flexDirection="column">
			<Box>
				{isReply ? <Text>Replay To: {messageContent}</Text> : <Text>Write A message:</Text>}
			</Box>
			<Box>
				<TextInput
					onSubmit={async (_) => {
						if (selectedUser) {
							if (isReply) {
								const newMessage = {
									content: message,
									media: null,
									isFromMe: true,
									id: Math.floor(Math.random() * 10000),
									sender: 'you',
									date: new Date(),
									isUnsupportedMessage: false
								} satisfies FormattedMessage;
								conversationStore.setState({ conversation: [...conversation, newMessage] });
								if (currentChatType === 'user') {
									sendMessage(client, selectedUser as UserInfo, message, true, messageAction.id);
								}
								setMessage('');
								setMessageAction(null);
								return;
							}
							messageAction?.action === 'edit' ? edit() : onSubmit(message);
							setMessage('');
						}
					}}
					placeholder="Write a message"
					value={message}
					onChange={async (value) => {
						setMessage(value);
						const chatConfig = getConfig('chat');
						if (chatConfig.sendTypingState) {
							if (currentChatType === 'user') {
								if (selectedUser) {
									await setUserTyping(client, selectedUser as UserInfo, currentChatType);
								}
							}
						}
					}}
					focus={isFocused}
				/>
			</Box>
		</Box>
	);
}
