import { getUserChats } from '@/telegram/client';
import { getConversationHistory, listenForUserMessages } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, render, Text, useFocus, useInput } from 'ink';
import React, { useEffect, useState } from 'react';
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
            {/* @ts-ignore */}
            <Box borderStyle="round" borderColor="green" flexDirection="row" minHeight={20} height={30}>
                {/* @ts-ignore */}
                <Box width={'30%'} flexDirection="column" borderRightstyle="round" borderRightColor="green">
                    <Sidebar />
                </Box>
                <ChatArea />
            </Box>
        </TGCliProvider>
    );
};

function Sidebar() {
    const { setSelectedUser, client, selectedUser } = useTGCli();
    const [activeChat, setActiveChat] = useState<ChatUser | null>(null);
    const [chatUsers, setChatUsers] = useState<ChatUser[]>([]);
    const { isFocused } = useFocus();

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
            listenForUserMessages(client, setChatUsers, selectedUser?.firstName || '');
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
                setActiveChat(nextUser);
            }
        } else if (key.downArrow) {
            const currentIndex = chatUsers.findIndex(({ peerId }) => peerId === activeChat?.peerId);
            const nextUser = chatUsers[currentIndex + 1];
            if (nextUser) {
                setActiveChat(nextUser);
            }
        }
    });

    return (
        // @ts-expect-error
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
            {chatUsers.map(({ firstName, peerId }) => (
                <Text color={activeChat?.peerId == peerId ? 'green' : 'white'} key={String(peerId)}>
                    {activeChat?.peerId == peerId && isFocused ? '>' : null} {firstName}
                </Text>
            ))}
        </Box>
    );
}

function ChatArea() {
    const { selectedUser } = useTGCli();
    const [conversation, setConversation] = useState<FormattedMessage[]>([]);

    const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
    const { isFocused } = useFocus();
    const selectedUserPeerID = String(selectedUser?.peerId);

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
        getConversationHistory(selectedUser).then(setConversation);
    }, [selectedUserPeerID]);

    return (
        <>
            {/* @ts-ignore */}
            <Box
                width={'70%'}
                height={'90%'}
                overflowY="visible"
                borderLeftColor={isFocused ? 'green' : ''}
                borderLeft={isFocused}
                flexDirection="column"
                gap={2}
            >
                <Text color="blue" bold>
                    {selectedUser?.firstName}
                </Text>
                {conversation ? (
                    conversation.map((message) => {
                        return (
                            // @ts-expect-error
                            <Box key={message.id} border width={'30%'}>
                                {activeMessage?.id == message.id && isFocused ? (
                                    <Text color={'green'}>{'>  '}</Text>
                                ) : null}
                                <Text>{message.content}</Text>
                            </Box>
                        );
                    })
                ) : (
                    <Text>No conversation</Text>
                )}
            </Box>
        </>
    );
}

export function initializeUI(client: TelegramClient) {
    render(<TGCli client={client} />);
}
