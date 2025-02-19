import { useTGCliStore } from '@/lib/store';
import { getUserChats } from '@/telegram/client';
import { listenForUserMessages } from '@/telegram/messages';
import { ChatUser, FormattedMessage } from '@/types';
import { Box, Text, useFocus, useInput } from 'ink';
import notifier from 'node-notifier';
import React, { useCallback, useEffect, useState } from 'react';
export function Sidebar() {
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