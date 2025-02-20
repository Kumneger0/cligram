#!/bin/env node
import { useTGCliStore } from '@/lib/store';
import { ChatArea } from '@/ui/chatArea';
import { Sidebar } from '@/ui/sidebar';
import { Box, render, Text, useFocus, useInput } from 'ink';
import React, { useEffect } from 'react';
import { TelegramClient } from 'telegram';
import { getConfig, setConfig } from './lib/utils/auth';

const HelpPage: React.FC = () => {
	const { isFocused } = useFocus({ autoFocus: true });

	return (
		<Box flexDirection="column" padding={1} borderStyle={isFocused ? "round" : undefined}
			borderColor={isFocused ? "green" : ""} >
			<Text>Key Bindings:</Text>
			<Box flexDirection='column'>
				<Box>
					<Text>Press 'q' to exit</Text>
				</Box>
				<Box>
					<Text>Press 'j' to go down</Text>
				</Box>
				<Box>
					<Text>Press 'k' to go up</Text>
				</Box>
				<Box>
					<Text>Press 'enter' to select chat</Text>
				</Box>
			</Box>
			<Box flexDirection='column'>
				<Text>Press 'h' to show this help</Text>
				<Text>Press 'c' to remove this help dialog</Text>
				<Text>Press 'x' to remove this help dialog and not show this message in the future</Text>
			</Box>
		</Box >
	);
};


const TGCli: React.FC<{ client: TelegramClient }> = ({ client: TelegramClient }) => {
	const selectedUser = useTGCliStore((state) => state.selectedUser);
	const updateClient = useTGCliStore((state) => state.updateClient);
	const config = getConfig()
	const [showHelp, setShowHelp] = React.useState((String(config?.skipHelp) === 'false') || !!!(config?.skipHelp));
    const client = useTGCliStore((state) => state.client);

	useInput((input, key) => {
		if (!showHelp) return;
		if (input === 'c') {
			setShowHelp(false);
		}
		if (input === 'x') {
			setShowHelp(false);
			setConfig('skipHelp', true)
		}
	})

	useEffect(() => {
		updateClient(TelegramClient);

	}, []);

	if (!client) return;
	if (showHelp) return <HelpPage />;
	return (<>
		<Box borderStyle="round" borderColor="green" flexDirection="row" minHeight={20} height={30}>
			<Box width={'30%'} flexDirection="column" borderRightColor="green">
				<Sidebar />
			</Box>
			<ChatArea key={selectedUser?.peerId.toString() ?? 'defualt-key'} />
		</Box>
	</>
	);
};


export async function initializeUI(client: TelegramClient) {
	render(<TGCli client={client} />);
}
