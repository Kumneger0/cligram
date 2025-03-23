import React, { useEffect, useState } from "react"
import { Box, Text, useFocus, useInput } from "ink"
import { useForwardMessageStore, useTGCliStore } from "@/lib/store"
import { getUserChats } from "@/telegram/client";
import { UserInfo } from "@/lib/types";
import { forwardMessage } from "@/telegram/messages";
import { componenetFocusIds } from "@/lib/utils/consts";
import chalk from "chalk";
function ForwardMessageModal({ width, height }: { width: number, height: number }) {
    const client = useTGCliStore((state) => state.client)
    const [chats, setChats] = useState<UserInfo[]>([])
    const { isFocused } = useFocus({ autoFocus: true, id: componenetFocusIds.forwardMessage })
    const bgColor = chalk.bgBlue(''.repeat(80));

    const [activeIndex, setActiveIndex] = useState(0)
    const setSelectedUser = useTGCliStore((state) => state.setSelectedUser)
    const forwardMessageOptions = useForwardMessageStore((state) => state.forwardMessageOptions)
    const setForwardMessageOptions = useForwardMessageStore((state) => state.setForwardMessageOptions)

    const [offset, setOffset] = useState(0)

    const setCurrentChatType = useTGCliStore((state) => state.setCurrentChatType)

    useEffect(() => {
        const getChats = async () => {
            if (!client) return;
            const chats = await getUserChats(client, 'PeerUser');
            setChats(chats)
        }
        getChats()
    }, [])


    useInput((input, key) => {
        if (!client) return;
        if (!forwardMessageOptions) return;
        if (key.upArrow || input === 'k') {
            setActiveIndex((prev) => Math.max(0, prev - 1));
            setOffset((prev) => Math.max(0, prev - 1))
        }
        if (key.downArrow || input === 'j') {
            const newOffset = Math.max(offset + 1, chats.length - forwardMessageModalHeight);
            if (newOffset < chats.length) {
                setOffset(newOffset)
            }
        }
        if (key.return) {
            const chat = chats[activeIndex]
            if (chat) {
                ; (async () => {
                    try {
                        setCurrentChatType('PeerUser')
                        setSelectedUser(chat)
                        await forwardMessage(client, {
                            fromPeer: {
                                peerId: forwardMessageOptions?.fromPeer.peerId,
                                accessHash: forwardMessageOptions?.fromPeer.accessHash
                            },
                            id: forwardMessageOptions?.id,
                            toPeer: {
                                peerId: chat.peerId,
                                accessHash: chat.accessHash
                            },
                            type: forwardMessageOptions?.type
                        })
                        setForwardMessageOptions(null)
                    } catch (err) {
                        console.error(err)
                        setForwardMessageOptions(null)
                    }
                })()
            }
        }
    })

    if (!client) return null
    if (!forwardMessageOptions) return null

    const modalBackadropWidth = width * (80 / 100);
    const modalBackadropHight = height * (80 / 100);

    const forwardMessageModalHeight = Math.floor(modalBackadropHight * 0.8)
    const forwardMessageModalWidth = Math.floor(modalBackadropWidth * 0.8)

    const visibleChats = chats.slice(offset, offset + forwardMessageModalHeight * 0.5)

    return (<>
        <Box
            borderColor={isFocused ? 'blue' : ''}
            borderStyle="round"
            flexDirection="column"
            width={modalBackadropWidth}
            height={modalBackadropHight}
            justifyContent="center"
            alignItems="center"
        >
            <Box position="absolute">
                <Text color="blue" backgroundColor="white">
                    {bgColor}
                </Text>
            </Box>
            <Box
                flexDirection="column"
                borderStyle="round"
                borderColor={'blue'}
                padding={1}
                width={forwardMessageModalWidth}
                height={forwardMessageModalHeight}
                alignItems="center"
                justifyContent="center"
                position="absolute"
                marginTop={5}
                marginLeft={30}
                marginRight={30}
            >
                <Box>
                    <Box display="flex" flexDirection="column" gap={1}>
                        <Text>
                            Select a chat to forward message to
                        </Text>
                        {visibleChats.map((chat, index) => (
                            <Text key={chat.accessHash.toString()} color={activeIndex === index ? 'green' : 'white'}>
                                {activeIndex === index ? '> ' : '  '}
                                {chat.firstName}
                            </Text>
                        ))}
                    </Box>
                </Box>
                <Box marginTop={2}>
                    <Text backgroundColor={'blue'} color={'white'}>
                        (Press ESC to close, Enter to forward)
                    </Text>
                </Box>
                <Box marginTop={1} flexDirection="column" alignItems="center">
                    <Text color="gray">Forward Navigation:</Text>
                    <Box gap={2}>
                        <Text color="red" bold>
                            j/k
                        </Text>
                        <Text color="green">navigate chats</Text>
                    </Box>
                </Box>
            </Box>
        </Box>
    </>

    )
}

export default ForwardMessageModal