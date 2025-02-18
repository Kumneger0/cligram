import { getUserChats } from '@/telegram/client';
import { deleteMessage, getAllMessages, listenForUserMessages, sendMessage } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import chalk from 'chalk';
import { Box, render, Text, useFocus, useInput, } from 'ink';
import Spinner from 'ink-spinner';
import TextInput from 'ink-text-input';
import notifier from 'node-notifier';
import React, { useCallback, useEffect, useState } from 'react';
import { TelegramClient } from 'telegram';
import { create } from 'zustand';

type MessageAction = {
    action: 'edit' | "delete" | 'reply',
    id: number
}
type TGCliStore = {
    client: TelegramClient | null,
    updateClient: (client: TelegramClient) => void,
    selectedUser: ChatUser | null;
    setSelectedUser: (selectedUser: ChatUser | null) => void;
    messageAction: MessageAction | null,
    setMessageAction: (messageAction: MessageAction) => void
}


const conversationStore = create<{ conversation: FormattedMessage[], setConversation: (conversation: FormattedMessage[]) => void }>((set) => ({
    conversation: [],
    setConversation: (conversation) => set({ conversation })
}))


const useTGCliStore = create<TGCliStore>((set) => (
    {
        client: null,
        updateClient: (client: TelegramClient) => set((state) => ({ ...state, client })),
        selectedUser: null,
        setSelectedUser: (selectedUser: ChatUser | null) => set((state) => ({ ...state, selectedUser })),
        messageAction: null,
        setMessageAction: (messageAction: MessageAction) => set((state) => ({ ...state, messageAction }))
    }))





const TGCli: React.FC<{ client: TelegramClient }> = ({ client: TelegramClient }) => {
    const selectedUser = useTGCliStore((state) => state.selectedUser);
    const updateClient = useTGCliStore((state) => state.updateClient);
    const client = useTGCliStore((state) => state.client);

    useEffect(() => {
        updateClient(TelegramClient)
    }, [])

    if (!client) return


    return (
        <Box borderStyle="round" borderColor="green" flexDirection="row" minHeight={20} height={30}>
            <Box width={'30%'} flexDirection="column" borderRightColor="green">
                <Sidebar />
            </Box>
            <ChatArea key={selectedUser?.peerId.toString() ?? 'defualt-key'} />
        </Box>
    );
};

function Sidebar() {
    const client = useTGCliStore((state) => state.client)!;
    const setSelectedUser = useTGCliStore((state) => state.setSelectedUser);


    const [activeChat, setActiveChat] = useState<ChatUser | null>(null);
    const [chatUsers, setChatUsers] = useState<(ChatUser & { unreadCount: number })[]>([]);
    const [offset, setOffset] = useState(0);

    const { isFocused } = useFocus();

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

    useEffect(() => {
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
            listenForUserMessages(client, onMessage);
        });
    }, []);

    useInput((_, key) => {
        if (!isFocused) return;
        if (key.return) {
            setSelectedUser(activeChat);
        }
        if (key.upArrow) {
            const currentIndex = chatUsers.findIndex(({ peerId }) => peerId === activeChat?.peerId);
            const nextUser = chatUsers[currentIndex - 1];
            if (nextUser) {
                setOffset((prev) => Math.max(prev - 1, 0));
                setActiveChat(nextUser);
            }
        } else if (key.downArrow) {
            const currentIndex = chatUsers.findIndex(({ peerId }) => peerId === activeChat?.peerId);
            const nextUser = chatUsers[currentIndex + 1];
            setOffset((prev) => Math.min(prev + 1, chatUsers.length - 50));
            if (nextUser) {
                setActiveChat(nextUser);
            }
        }
    });

    const visibleChats = chatUsers.slice(offset, offset + 50);

    return (
        <Box
            width={'40%'}
            flexDirection="column"
            height={'100%'}
            borderStyle={isFocused ? 'round' : undefined}
            borderColor={isFocused ? 'green' : ''}
        >
            <Text color="blue" bold>
                Chats
            </Text>
            {visibleChats.map(({ firstName, peerId, unreadCount }) => (
                <Text color={activeChat?.peerId == peerId ? 'green' : 'white'} key={String(peerId)}>
                    {activeChat?.peerId == peerId && isFocused ? '>' : null} {firstName}{' '}
                    {unreadCount > 0 && <Text color="red">({unreadCount})</Text>}
                </Text>
            ))}
        </Box>
    );
}

function ChatArea() {
    const selectedUser = useTGCliStore((state) => state.selectedUser);
    const selectedUserPeerID = String(selectedUser?.peerId);
    const client = useTGCliStore((state) => state.client)!;
    const { conversation, setConversation } = conversationStore((state) => state);



    const [offsetId, setOffsetId] = useState<number | undefined>(undefined);
    const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const setMessageAction = useTGCliStore((state) => state.setMessageAction);
    const { isFocused } = useFocus();

    const [offset, setOffset] = useState(0);


    useInput((input, key) => {
        if (!isFocused) return;

        if (input === 'd') {
            setMessageAction({ action: 'delete', id: activeMessage?.id! })
            setIsModalOpen(true)
        }
        if (key.return) {
            //TODO: do something with the message
            setIsModalOpen(true);
        }
        if (key.upArrow) {
            const currentIndex = conversation?.findIndex(({ id }) => id === activeMessage?.id);
            const nextMessage = conversation[currentIndex - 1];
            if (nextMessage) {
                setActiveMessage(nextMessage);
            }
        } else if (key.downArrow) {
            const currentIndex = conversation?.findIndex(({ id }) => id === activeMessage?.id);
            const nextMessage = conversation[currentIndex + 1];
            if (nextMessage) {
                setActiveMessage(nextMessage);
            }
        }
    });

    useEffect(() => {
        if (!selectedUser) return;
        setIsLoading(true);
        getAllMessages(client, selectedUser).then(async (conversation) => {
            setConversation(conversation);
            setOffsetId(conversation?.[0]?.id);
            setIsLoading(false);
        });
        listenForUserMessages(client, (message) => {
            const from = message.sender;
            if (from === selectedUser?.firstName) {
                setConversation([...conversation, message]);
                setOffsetId(message.id);
            }
        });
    }, [selectedUserPeerID]);

    useInput((_, key) => {
        if (!isFocused) return;

        if (key.downArrow) {
            setOffset((prev) => Math.min(prev + 1, conversation.length - 20));
            const atEnd = offset === conversation.length - 20;
            if (atEnd && selectedUser) {
                const appendMessages = async () => {
                    const newMessages = await getAllMessages(client, selectedUser, offsetId);
                    const updatedConversation = [...conversation, ...newMessages];
                    setConversation(
                        updatedConversation.filter(
                            ({ id }, i) => updatedConversation.findIndex((c) => c.id == id) === i
                        )
                    );
                    setOffsetId(newMessages?.[0]?.id);
                };
                appendMessages();
            }
        } else if (key.upArrow) {
            setOffset((prev) => Math.max(prev - 1, 0));
        }
    });

    const visibleMessages = conversation.slice(offset, offset + 50);

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

    return (
        <>
            {isModalOpen && (
                <Modal onClose={() => setIsModalOpen(false)}>
                    <Text>Hello</Text>
                </Modal>
            )}

            {!isModalOpen &&
                <Box flexDirection="column" height="100%" width={'70%'}>
                    <Box
                        width={'100%'}
                        height={'90%'}
                        overflowY="hidden"
                        borderStyle={isFocused ? 'classic' : undefined}
                        flexDirection="column"
                        gap={1}
                        paddingLeft={2}
                    >
                        <Text color="blue" bold>
                            {selectedUser?.firstName}
                        </Text>
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
                                            {message.media && <Text>{message.media}</Text>}
                                        </Text>
                                        <Text wrap="wrap" color={'white'} backgroundColor={activeMessage?.id == message.id && isFocused ? "blue" : ""}>{message.content}</Text>
                                    </Box>
                                </Box>
                            );
                        })}
                    </Box>
                    <MessageInput
                        onSubmit={(message) => {
                            if (selectedUser) {
                                const newMessage = {
                                    content: message,
                                    media: null,
                                    isFromMe: true,
                                    id: Math.floor(Math.random() * 10000),
                                    sender: 'you'
                                } satisfies FormattedMessage;
                                setConversation([...conversation, newMessage]);
                                sendMessage(
                                    client,
                                    { peerId: selectedUser.peerId, accessHash: selectedUser.accessHash },
                                    message
                                );
                            }
                        }}
                    />
                </Box>
            }

        </>
    );
}

const messageActions = [
    {
        name: 'delete',
        description: 'are u sure you want to delte',
        deleteMessageShortCuts: {
            'delete': 'y',
            //TODO: allow user to chose delete only for him or for everyone
        },
        action: async (client: TelegramClient, messageId: number, selectedUser: ChatUser) => {
            await deleteMessage(client, selectedUser, messageId)
        }
    },

] as const

const Modal: React.FC<{ onClose: () => void; children: React.ReactNode }> = ({
    onClose,
}) => {
    const { isFocused } = useFocus({ autoFocus: true });
    const client = useTGCliStore((state) => state.client)!;
    const selectedUser = useTGCliStore((state) => state.selectedUser);
    const msgAction = useTGCliStore((state) => state.messageAction);
    const messageAction = useTGCliStore((state) => state.messageAction);
    const messageActionCurrentActiveKey = messageAction?.action;
    const { action, deleteMessageShortCuts, description } = messageActions.find(({ name }) => name === messageActionCurrentActiveKey)!;
    const { conversation, setConversation } = conversationStore((state) => state);

    useInput(async (_, key) => {
        if (key.escape) {
            onClose()
            return
        }
        const messageId = msgAction?.id
        if (!messageId || !selectedUser) {
            console.log(messageId, selectedUser)
            return
        }
        action(client, messageId, selectedUser)
        const filterConversation = conversation.filter(({ id }) => id !== messageId)
        setConversation(filterConversation)
        onClose()
    })


    const bgColor = chalk.bgBlue(''.repeat(80));

    return (
        <Box borderColor={isFocused ? 'blue' : ""} borderStyle="round" flexDirection="column" width={80} height={20} justifyContent="center" alignItems="center">
            <Box position='absolute'>
                <Text color="blue" backgroundColor="white">{bgColor}</Text>
            </Box>
            <Box
                flexDirection="column"
                borderStyle="round"
                borderColor={'blue'}
                padding={1}
                width={50}
                alignItems="center"
                justifyContent="center"
                position='absolute'
                marginTop={15} marginLeft={30} marginRight={30}
            >
                <Text color="blue" bold>
                    {description}
                </Text>
                {Object.keys(deleteMessageShortCuts).map((key) => {
                    return <Box key={key} gap={2}>
                        <Text color="red" bold>
                            {key}
                        </Text>
                        <Text color="green">
                            {deleteMessageShortCuts[key as keyof typeof deleteMessageShortCuts]}
                        </Text>
                    </Box>
                })}
                <Box>
                    <Text backgroundColor={'blue'} color={'white'}>
                        (Press ESC to close)
                    </Text>
                </Box>
            </Box>
        </Box>
    );
};

function MessageInput({ onSubmit }: { onSubmit: (message: string) => void }) {
    const [message, setMessage] = useState('');

    const selectedUser = useTGCliStore((state) => state.selectedUser);
    const { isFocused } = useFocus();

    return (
        <Box borderStyle={isFocused ? 'classic' : undefined}>
            <Box marginRight={1}>
                <Text>Write A message:</Text>
            </Box>

            <TextInput
                onSubmit={async (_) => {
                    if (selectedUser) {
                        onSubmit(message);
                        setMessage('');
                    }
                }}
                placeholder="Write a message"
                value={message}
                onChange={setMessage}
                focus={isFocused}
            />
        </Box>
    );
}

export function initializeUI(client: TelegramClient) {
    render(<TGCli client={client} />);
}
