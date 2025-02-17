import { getUserChats } from '@/telegram/client';
import { getAllMessages, listenForUserMessages, sendMessage } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, render, Text, useFocus, useInput } from 'ink';
import Spinner from 'ink-spinner';
import TextInput from 'ink-text-input';
import React, { ComponentRef, useCallback, useEffect, useRef, useState } from 'react';
import { TelegramClient } from 'telegram';

type TGCliContextType = {
    client: TelegramClient;
    selectedUser: ChatUser | null;
    setSelectedUser: React.Dispatch<React.SetStateAction<ChatUser | null>>;
};

const TGCliContext = React.createContext<TGCliContextType | null>(null);

export const TGCliProvider = ({
    children,
    ...rest
}: TGCliContextType & { children: React.ReactNode }) => {
    return <TGCliContext.Provider value={{ ...rest }}>{children}</TGCliContext.Provider>;
};

export const useTGCli = () => {
    const client = React.useContext(TGCliContext);
    if (!client) {
        throw new Error('useTGCli must be used within a TGCliProvider');
    }
    return client;
};

const TGCli: React.FC<{ client: TelegramClient }> = ({ client }) => {
    const [selectedUser, setSelectedUser] = useState<ChatUser | null>(null);

    return (
        <TGCliProvider client={client} selectedUser={selectedUser} setSelectedUser={setSelectedUser}>
            <Box borderStyle="round" borderColor="green" flexDirection="row" minHeight={20} height={30}>
                <Box width={'30%'} flexDirection="column" borderRightColor="green">
                    <Sidebar />
                </Box>
                <ChatArea key={selectedUser?.peerId.toString() ?? 'defualt-key'} />
            </Box>
        </TGCliProvider>
    );
};

function Sidebar() {
    const { setSelectedUser, client } = useTGCli();
    const [activeChat, setActiveChat] = useState<ChatUser | null>(null);
    const [chatUsers, setChatUsers] = useState<(ChatUser & { unreadCount: number })[]>([]);
    const [offset, setOffset] = useState(0);

    const { isFocused } = useFocus();

    const onMessage = useCallback((message: Partial<FormattedMessage>) => {
        const sender = message.sender;
        const content = message.content;
        const isFromMe = message.isFromMe;
        setChatUsers((prev) => prev.map((user) => {
            if (user.firstName === sender) {
                return {
                    ...user,
                    unreadCount: user.unreadCount + 1,
                    lastMessage: content,
                    isFromMe
                };
            }
            return user;
        }));
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
            listenForUserMessages(client, onMessage)
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
                    {activeChat?.peerId == peerId && isFocused ? '>' : null} {firstName} {unreadCount > 0 && <Text color="red">({unreadCount})</Text>}
                </Text>
            ))}
        </Box>
    );
}

function ChatArea() {
    const { selectedUser, client } = useTGCli();
    const [conversation, setConversation] = useState<FormattedMessage[]>([]);
    const [offsetId, setOffsetId] = useState<number | undefined>(undefined);
    const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const coversationRef = useRef<ComponentRef<typeof Box>>(null);


    const { isFocused } = useFocus();
    const selectedUserPeerID = String(selectedUser?.peerId);

    const [offset, setOffset] = useState(0);

    useInput((_, key) => {
        if (!isFocused) return;
        if (key.return) {
            //TODO: do something with the message
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
            <Text>
                <Text color="green">
                    <Spinner type="dots" />
                </Text>
                {' Loading conversations...'}
            </Text>
        );
    }

    return (
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
                            ref={coversationRef}
                            alignSelf={message.isFromMe ? 'flex-end' : 'flex-start'}
                            key={message.id}
                            width={'30%'}
                            height={'auto'}
                            flexShrink={0}
                            flexGrow={0}
                        >
                            {activeMessage?.id == message.id && isFocused ? (
                                <Text color={'green'}>{'>  '}</Text>
                            ) : null}
                            <Text
                                backgroundColor={activeMessage?.id === message.id && isFocused ? 'blue' : ''}
                                color={activeMessage?.id === message.id && isFocused ? 'white' : ''}
                            >
                                {message.content}
                            </Text>
                        </Box>
                    );
                })}
            </Box>
            <MessageInput />
        </Box>
    );
}

function MessageInput() {
    const [message, setMessage] = useState('');
    const { client, selectedUser } = useTGCli();
    const { isFocused } = useFocus();

    return (
        <Box borderStyle={isFocused ? 'classic' : undefined}>
            <Box marginRight={1}>
                <Text>Write A message:</Text>
            </Box>

            <TextInput
                onSubmit={async (_) => {
                    if (selectedUser) {
                        await sendMessage(
                            client,
                            { peerId: selectedUser.peerId, accessHash: selectedUser.accessHash },
                            message
                        );
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
