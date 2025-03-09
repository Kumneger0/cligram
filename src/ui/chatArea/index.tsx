import { conversationStore, useTGCliStore } from '@/lib/store';
import { formatLastSeen } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import { editMessage, getAllMessages, listenForEvents, sendMessage } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, Text, useFocus, useFocusManager, useInput } from 'ink';
import Spinner from 'ink-spinner';
import TextInput from 'ink-text-input';
import React, { useEffect, useLayoutEffect, useState } from 'react';
import { Modal } from '../modal/Modal';
import { Fragment } from 'react';
import { ChannelInfo } from '@/telegram/client';
const formatDate = (date: Date) =>
	date.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		year: 'numeric'
	});

const groupMessagesByDate = (messages: FormattedMessage[]) => {
	const sortedMessages = messages.sort((a, b) => a.date.getTime() - b.date.getTime());

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
	const selectedUser = useTGCliStore((state) => state.selectedUser);
	const client = useTGCliStore((state) => state.client)!;
	const { conversation, setConversation } = conversationStore((state) => state);
	const setMessageAction = useTGCliStore((state) => state.setMessageAction);
	const [offsetId, setOffsetId] = useState<number | undefined>(undefined);

	const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
	const [isLoading, setIsLoading] = useState(false);
	const [isModalOpen, setIsModalOpen] = useState(false);
	const [offset, setOffset] = useState(0);

	const conversationAreaHieght = height * (70 / 100);

	const currentChatType = useTGCliStore((state) => state.currentChatType);
	const currentlySelectedChatId =
		currentChatType === 'PeerUser'
			? (selectedUser as ChatUser)?.peerId
			: (selectedUser as ChannelInfo)?.channelId;

	const selectedUserPeerID = String(currentlySelectedChatId);

	useEffect(() => {
		if (!selectedUser) return;
		setIsLoading(true);
		let unsubscribe: () => void;
		const id =
			currentChatType === 'PeerUser'
				? (selectedUser as ChatUser).peerId
				: ((selectedUser as ChannelInfo).channelId as unknown as bigInt.BigInteger);
		const accessHash =
			currentChatType === 'PeerUser'
				? (selectedUser as ChatUser).accessHash
				: ((selectedUser as ChannelInfo).accessHash as unknown as bigInt.BigInteger);

		(async () => {
			const conversation = await getAllMessages(
				{
					client,
					peerInfo: {
						accessHash,
						peerId: id,
						userFirtNameOrChannelTitle:
							currentChatType === 'PeerUser'
								? (selectedUser as ChatUser).firstName
								: (selectedUser as ChannelInfo).title,
					},
					chatAreaWidth: width
				},
				currentChatType
			);
			setConversation(conversation);
			setOffsetId(conversation?.[0]?.id);
			setIsLoading(false);
			setActiveMessage(conversation.at(-1) ?? null);

			if (currentChatType === 'PeerUser') {
				unsubscribe = await listenForEvents(client, {
					onMessage: (message) => {
						const from = message.sender;
						if (from === (selectedUser as ChatUser).firstName) {
							setConversation([...conversation, message]);
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

	const visibleMessages = conversation.slice(offset, offset + conversationAreaHieght);

	useInput(async (input, key) => {
		if (!isFocused) return;

		if (input === 'd') {
			setMessageAction({ action: 'delete', id: activeMessage?.id! });
			setIsModalOpen(true);
			return;
		}

		if (input === 'e') {
			if (!activeMessage?.isFromMe) return;
			setMessageAction({ action: 'edit', id: activeMessage?.id! });
			focus(componenetFocusIds.messageInput);
			return;
		}

		if (input === 'r') {
			setMessageAction({ action: 'reply', id: activeMessage?.id! });
			focus(componenetFocusIds.messageInput);
			return;
		}



		if (key.upArrow || input == 'k') {
			const currentIndex = conversation?.findIndex(({ id }) => id === activeMessage?.id);
			let nextMessage = conversation[currentIndex - 1];
			if (offset == 0 && selectedUser) {
				const appendMessages = async () => {
					const peerId = currentChatType === 'PeerUser' ? (selectedUser as ChatUser).peerId : (selectedUser as ChannelInfo).channelId;
					const accessHash = currentChatType === 'PeerUser' ? (selectedUser as ChatUser).accessHash : (selectedUser as ChannelInfo).accessHash;
					const userFirtNameOrChannelTitle = currentChatType === 'PeerUser' ? (selectedUser as ChatUser).firstName : (selectedUser as ChannelInfo).title;
					const newMessages = await getAllMessages({
						client,
						peerInfo: {
							accessHash: accessHash as unknown as bigInt.BigInteger,
							peerId: peerId as unknown as bigInt.BigInteger,
							userFirtNameOrChannelTitle
						},
						offsetId,
						chatAreaWidth: width
					}, currentChatType);
					const updatedConversation = [...newMessages, ...conversation];
					setConversation(
						updatedConversation.filter(
							({ id }, i) => updatedConversation.findIndex((c) => c.id == id) === i
						)
					);
					nextMessage = updatedConversation[currentIndex - 1];
					setOffsetId(newMessages?.[0]?.id);
					setOffset(newMessages.length);
				};
				await appendMessages();
				return;
			}
			if (nextMessage) {
				setActiveMessage(nextMessage);
			}
			setOffset((prev) => Math.max(prev - 1, 0));
		} else if (key.downArrow || input === 'j') {
			const currentIndex = conversation?.findIndex(({ id }) => id === activeMessage?.id);
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
		currentChatType === 'PeerUser'
			? (selectedUser as ChatUser)?.lastSeen
				? formatLastSeen((selectedUser as ChatUser)?.lastSeen!)
				: 'Unknown'
			: '';

	const groupedMessages = groupMessagesByDate(visibleMessages);



	return (
		<>
			{isModalOpen && <Modal onClose={() => setIsModalOpen(false)} />}
			{!isModalOpen && (
				<Box flexDirection="column" height={height} width={width}>
					<Box gap={1}>
						<Text color="blue" bold>
							{currentChatType === 'PeerUser'
								? (selectedUser as ChatUser)?.firstName
								: (selectedUser as ChannelInfo)?.title}
						</Text>
						<Text>
							{currentChatType === 'PeerUser'
								? (selectedUser as ChatUser)?.isOnline
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
											<Box
												alignSelf={message.isFromMe ? 'flex-end' : 'flex-start'}
												key={message.id}
												width={'30%'}
												height={'auto'}
												flexShrink={0}
												flexDirection="column"
												flexGrow={0}
											>
												{activeMessage?.id == message.id && isFocused ? (
													<Text color={'green'}>{'>  '}</Text>
												) : null}
												<Box flexDirection="column">
													<Text
														backgroundColor={
															activeMessage?.id === message.id && isFocused ? 'blue' : ''
														}
														color={activeMessage?.id === message.id && isFocused ? 'white' : ''}
													>
														{message.media && <Text wrap="end">{message.media}</Text>}
													</Text>
													<Text
														wrap="wrap"
														color={'white'}
														bold={!!message.document}
														underline={!!message.document}
														backgroundColor={
															activeMessage?.id == message.id && isFocused ? 'blue' : ''
														}
													>
														{message.content}
													</Text>
													<Text>
														{date.toLocaleTimeString([], {
															hour: '2-digit',
															minute: '2-digit',
															hour12: true
														})}
													</Text>
												</Box>
											</Box>
										);
									})}
								</Fragment>
							);
						})}
					</Box>
					{(currentChatType === 'PeerUser' ||
						(currentChatType === 'PeerChannel' && (selectedUser as ChannelInfo)?.isCreator)) && (
						<MessageInput
							onSubmit={async (message) => {
								if (selectedUser) {
									const newMessage = {
										content: message,
										media: null,
										isFromMe: true,
										id: Math.floor(Math.random() * 10000),
										sender: 'you',
										date: new Date()
									} satisfies FormattedMessage;
									setConversation([...conversation, newMessage]);
									const id = currentChatType === 'PeerUser'
										? (selectedUser as ChatUser).peerId
										: (selectedUser as ChannelInfo).channelId;
									const accessHash = currentChatType === 'PeerUser'
										? (selectedUser as ChatUser).accessHash
										: (selectedUser as ChannelInfo).accessHash;
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

function MessageInput({ onSubmit }: { onSubmit: (message: string) => void }) {
	const [message, setMessage] = useState('');
	const selectedUser = useTGCliStore((state) => state.selectedUser);
	const { isFocused } = useFocus({ id: componenetFocusIds.messageInput });
	const messageAction = useTGCliStore((state) => state.messageAction);
	const client = useTGCliStore((state) => state.client)!;
	const setMessageAction = useTGCliStore((state) => state.setMessageAction);
	const conversation = conversationStore((state) => state.conversation);
	const messageContent = conversation.find(({ id }) => id === messageAction?.id)?.content;
	const isReply = messageAction?.action === 'reply';

	const currentChatType = useTGCliStore((state) => state.currentChatType);

	useLayoutEffect(() => {
		if (isReply) return;
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
				date: new Date()
			} satisfies FormattedMessage;
			const updatedConversation = conversation.map((msg) => {
				if (msg.id === messageAction?.id) {
					return newMessage;
				}
				return msg;
			});
			conversationStore.setState({ conversation: updatedConversation });
			if (currentChatType === 'PeerUser') {
				editMessage(client, selectedUser as ChatUser, messageAction?.id!, message);
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
									date: new Date()
								} satisfies FormattedMessage;
								conversationStore.setState({ conversation: [...conversation, newMessage] });
								if (currentChatType === 'PeerUser') {
									sendMessage(client, selectedUser as ChatUser, message, true, messageAction?.id);
								}
								setMessage('');
								setMessageAction(null);
								return;
							}
							messageAction?.action == 'edit' ? edit() : onSubmit(message);
							setMessage('');
						}
					}}
					placeholder="Write a message"
					value={message}
					onChange={setMessage}
					focus={isFocused}
				/>
			</Box>
		</Box>
	);
}
