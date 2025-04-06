import { conversationStore, useForwardMessageStore, useTGCliStore } from '@/lib/store';
import { ChatArea } from '@/ui/chatArea';
import { Sidebar } from '@/ui/sidebar';
import { Box, Instance, render, Text, useFocus, useInput, useStdout } from 'ink';
import React, { useEffect, useState } from 'react';
import { TelegramClient } from 'telegram';
import { ChannelInfo, UserInfo } from './lib/types';
import { onMessage, onUserOnlineStatus } from './lib/utils';
import { getConfig, setConfig } from './lib/utils/auth';
import { getUserChats } from './telegram/client';
import { listenForEvents } from './telegram/messages';
import { SearchModal } from './ui/Search';
import ShowKeyBinding from './ui/ShowKeyBinding';
import ForwardMessageModal from './ui/forwardMessage';
const HelpPage: React.FC = () => {
	const { isFocused } = useFocus({ autoFocus: true });

	return (
		<Box
			flexDirection="column"
			padding={1}
			borderStyle={isFocused ? 'round' : undefined}
			borderColor={isFocused ? 'green' : ''}
		>
			<Text>Key Bindings:</Text>
			<Box flexDirection="column">
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
			<Box flexDirection="column">
				<Text>Press 'h' to show this help</Text>
				<Text>Press 'c' to remove this help dialog</Text>
				<Text>Press 'x' to remove this help dialog and not show this message in the future</Text>
			</Box>
		</Box>
	);
};

const TGCli: React.FC<{ client: TelegramClient }> = ({ client: telegramClient }) => {
	const selectedUser = useTGCliStore((state) => {
		return state.selectedUser;
	});

	const getSelectedUser = useTGCliStore((state) => {
		return state.getSelectedUser;
	});

	const updateClient = useTGCliStore((state) => {
		return state.updateClient;
	});
	const currentlyFocused = useTGCliStore((state) => {
		return state.currentlyFocused;
	});

	const forwardMessageOptions = useForwardMessageStore((state) => {
		return state.forwardMessageOptions;
	});

	const setSelectedUser = useTGCliStore((state) => {
		return state.setSelectedUser;
	});

	const { updateConversations } = conversationStore((state) => {
		return state;
	});

	const [userChats, setUserChats] = useState<Awaited<ReturnType<typeof getUserChats>>>();

	const config = getConfig();
	const [showHelp, setShowHelp] = React.useState(
		String(config?.skipHelp) === 'false' || !!!config?.skipHelp
	);
	const currentChatType = useTGCliStore((state) => {
		return state.currentChatType;
	});
	const client = useTGCliStore((state) => {
		return state.client;
	});

	const searchMode = useTGCliStore((state) => {
		return state.searchMode;
	});

	const { stdout } = useStdout();
	const [size, setSize] = useState({
		columns: stdout.columns,
		rows: stdout.rows
	});

	useEffect(() => {
		const handleResize = () => {
			setSize({
				columns: stdout.columns,
				rows: stdout.rows
			});
		};

		stdout.on('resize', handleResize);
		return () => {
			stdout.off('resize', handleResize);
		};
	}, [stdout]);

	useInput((input, _key) => {
		if (!showHelp) {
			return;
		}
		if (input === 'c') {
			setShowHelp(false);
		}
		if (input === 'x') {
			setShowHelp(false);
			setConfig('skipHelp', true);
		}
	});

	useEffect(() => {
		updateClient(telegramClient);
		let unsubscribe: (() => void) | undefined;
		const getChats = async () => {
			const result = await getUserChats(telegramClient, currentChatType);
			return result;
		};
		getChats().then(async (userChats) => {
			unsubscribe = await listenForEvents(telegramClient, {
				onMessage: (message, user) => {
					onMessage(message, userChats, currentChatType, user, setUserChats);
					const selectedUser = getSelectedUser();
					if (currentChatType === 'user') {
						if (user?.firstName === (selectedUser as UserInfo).firstName) {
							updateConversations([message]);
						}
					}
				},
				onUserOnlineStatus: (user) => {
					onUserOnlineStatus({
						user,
						currentChatType,
						selectedUser,
						setSelectedUser,
						setUserChats
					});
				}
			});
			setUserChats(userChats);
			if (!selectedUser) {
				setSelectedUser(userChats.dialogs?.[0] ?? null);
			}
		});
		return () => {
			setUserChats({ dialogs: [], lastDialog: null });
			return unsubscribe?.();
		};
	}, [currentChatType]);

	if (!client) {
		return;
	}

	if (showHelp) {
		return <HelpPage />;
	}

	const sidebarWidth = size.columns * (30 / 100);
	const chatAreaWidth = size.columns - sidebarWidth;
	const height = size.rows - 1;

	const currentlySelectedChatId =
		currentChatType === 'user'
			? (selectedUser as UserInfo | null)?.peerId
			: (selectedUser as ChannelInfo | null)?.channelId;

	let ComponentToRender: React.FC<{ height: number; width: number }> = ChatArea;

	if (!!forwardMessageOptions) {
		ComponentToRender = ForwardMessageModal;
	}
	if (!!searchMode) {
		ComponentToRender = SearchModal;
	}
	if (!userChats) {
		return <Text>Loading...</Text>;
	}

	return (
		<>
			<Box
				borderStyle="round"
				borderColor="green"
				flexDirection="row"
				minHeight={height}
				height={height}
				width={size.columns}
			>
				<Box width={sidebarWidth} flexDirection="column" borderRightColor="green">
					<Sidebar
						userChats={userChats}
						key={currentChatType}
						height={height}
						width={sidebarWidth}
					/>
				</Box>
				<ComponentToRender
					key={(currentlySelectedChatId ?? 'default-key').toString()}
					height={height}
					width={chatAreaWidth}
				/>
			</Box>
			<ShowKeyBinding type={currentlyFocused || 'general'} />
		</>
	);
};

export async function initializeUI(client: TelegramClient): Promise<Instance> {
	const root = render(<TGCli client={client} />);
	return root;
}
