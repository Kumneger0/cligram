import { conversationStore, useTGCliStore } from '@/lib/store';
import { formatLastSeen } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import {
    editMessage,
    getAllMessages,
    listenForEvents,
    sendMessage
} from '@/telegram/messages';
import { FormattedMessage } from '@/types';
import { Box, Text, useFocus, useFocusManager, useInput } from 'ink';
import Spinner from 'ink-spinner';
import TextInput from 'ink-text-input';
import React, { useEffect, useLayoutEffect, useState } from 'react';
import { Modal } from '../modal/Modal';

export function ChatArea({ height, width }: { height: number, width: number }) {
    const selectedUser = useTGCliStore((state) => state.selectedUser);
    const selectedUserPeerID = String(selectedUser?.peerId);
    const client = useTGCliStore((state) => state.client)!;
    const { conversation, setConversation } = conversationStore((state) => state);
    const [offsetId, setOffsetId] = useState<number | undefined>(undefined);
    const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const setMessageAction = useTGCliStore((state) => state.setMessageAction);
    const { isFocused } = useFocus({ id: componenetFocusIds.chatArea });
    const { focus } = useFocusManager();
    const [offset, setOffset] = useState(0);
    const conversationAreaHieght = height * (70 / 100)

    useEffect(() => {
        if (!selectedUser) return;
        setIsLoading(true);
        getAllMessages({ client, user: selectedUser, chatAreaWidth: width }).then(async (conversation) => {
            setConversation(conversation);
            setOffsetId(conversation?.[0]?.id);
            setIsLoading(false);
            setActiveMessage(conversation.at(-1) ?? null);
        });
        listenForEvents(client, {
            onMessage: (message) => {
                const from = message.sender;
                if (from === selectedUser?.firstName) {
                    setConversation([...conversation, message]);
                    setOffsetId(message.id);
                    setActiveMessage(message);
                }
            }
        });
    }, [selectedUserPeerID]);


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
                    const newMessages = await getAllMessages({client, user:selectedUser, offsetId, chatAreaWidth:width});
                    const updatedConversation = [...newMessages, ...conversation,];
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
                return
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

    const selectedUserLastSeen = selectedUser?.lastSeen ? formatLastSeen(selectedUser?.lastSeen) : 'Unknown';

    return (
        <>
            {isModalOpen && (
                <Modal onClose={() => setIsModalOpen(false)}>
                    <Text>Hello</Text>
                </Modal>
            )}

            {!isModalOpen && (
                <Box flexDirection="column" height={height} width={width}>
                    <Box gap={1}>
                        <Text color="blue" bold>
                            {selectedUser?.firstName}
                        </Text>
                        <Text>
                            {selectedUser?.isOnline ? 'Online' : `${selectedUserLastSeen}`}
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

                        {visibleMessages.map((message) => {

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
                                            backgroundColor={activeMessage?.id === message.id && isFocused ? 'blue' : ''}
                                            color={activeMessage?.id === message.id && isFocused ? 'white' : ''}
                                        >
                                            {/* {message.media && <Text wrap='end'>{message.media}</Text>} */}
                                        </Text>
                                        <Text
                                            wrap="wrap"
                                            color={'white'}
                                            backgroundColor={activeMessage?.id == message.id && isFocused ? 'blue' : ''}
                                        >
                                            {message.content}
                                        </Text>
                                    </Box>
                                </Box>
                            );
                        })}
                    </Box>
                    <MessageInput
                        onSubmit={async (message) => {
                            if (selectedUser) {
                                const newMessage = {
                                    content: message,
                                    media: null,
                                    isFromMe: true,
                                    id: Math.floor(Math.random() * 10000),
                                    sender: 'you'
                                } satisfies FormattedMessage;
                                setConversation([...conversation, newMessage]);
                                await sendMessage(
                                    client,
                                    { peerId: selectedUser.peerId, accessHash: selectedUser.accessHash },
                                    message
                                );
                            }
                        }}
                    />
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
                sender: 'you'
            } satisfies FormattedMessage;
            const updatedConversation = conversation.map((msg) => {
                if (msg.id === messageAction?.id) {
                    return newMessage;
                }
                return msg;
            });
            conversationStore.setState({ conversation: updatedConversation });
            editMessage(client, selectedUser, messageAction?.id!, message);
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
                                    sender: 'you'
                                } satisfies FormattedMessage;
                                conversationStore.setState({ conversation: [...conversation, newMessage] });
                                sendMessage(client, selectedUser, message, true, messageAction?.id);
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
