import { getUserChats } from '@/telegram/client';
import { getConversationHistory, listenForUserMessages, sendMessage } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, render, Text, useFocus, useInput } from 'ink';
import TextInput from 'ink-text-input';
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
            <Box borderStyle="round" borderColor="green" flexDirection="row" minHeight={20} height={40}>
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
    const [offset, setOffset] = useState(0);

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
            {visibleChats.map(({ firstName, peerId }) => (
                <Text color={activeChat?.peerId == peerId ? 'green' : 'white'} key={String(peerId)}>
                    {activeChat?.peerId == peerId && isFocused ? '>' : null} {firstName}
                </Text>
            ))}
        </Box>
    );
}

function ChatArea() {
    const { selectedUser, client } = useTGCli();
    const [conversation, setConversation] = useState<FormattedMessage[]>([]);

    const [activeMessage, setActiveMessage] = useState<FormattedMessage | null>(null);
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
        getConversationHistory(client, selectedUser).then(setConversation);
    }, [selectedUserPeerID]);

    useInput((_, key) => {
        if (!isFocused) return;
        if (key.downArrow) {
            setOffset((prev) => Math.min(prev + 1, conversation.length - 50));
        } else if (key.upArrow) {
            setOffset((prev) => Math.max(prev - 1, 0));
        }
    });

    const visibleMessages = conversation.slice(offset, offset + 50);

    return (
        //   @ts-ignore 
        <Box flexDirection='column' height='100%'
            width={'70%'}
        >
            {/* @ts-ignore */}
            <Box
                width={'100%'}
                height={'90%'}
                overflowY="hidden"
                borderStyle={isFocused ? 'classic' : undefined}
                flexDirection="column"
                gap={2}
                paddingLeft={2}
            >
                <Text color="blue" bold>
                    {selectedUser?.firstName}
                </Text>
                {visibleMessages ? (
                    visibleMessages.map((message) => {
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
            <MessageInput />
        </Box>
    );
}



function MessageInput() {
    const [message, setMessage] = useState('');
    const { client, selectedUser } = useTGCli()
    const { isFocused } = useFocus();

    return (
        // @ts-expect-error
        <Box>
            {/* @ts-ignore */}
            <Box marginRight={1}>
                <Text>Write A message:</Text>
            </Box>

            <TextInput onSubmit={async (_) => {
                if (selectedUser) {
                    await sendMessage(client, { peerId: selectedUser.peerId, accessHash: selectedUser.accessHash }, message)
                    setMessage('')
                }
            }} placeholder='Write a message' value={message} onChange={setMessage} focus={isFocused} />
        </Box>
    );
}





export function initializeUI(client: TelegramClient) {
    render(<TGCli client={client} />);
}
