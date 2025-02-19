import { useTGCliStore } from '@/lib/store';
import { Box, render } from 'ink';
import React, { useEffect } from 'react';
import { TelegramClient } from 'telegram';
import { ChatArea } from '@/ui/chatArea'
import { Sidebar } from '@/ui/sidebar'

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


export function initializeUI(client: TelegramClient) {
    render(<TGCli client={client} />);
}