import React from 'react';
import { conversationStore, useTGCliStore } from '@/lib/store';
import { deleteMessage } from '@/telegram/messages';
import { Box, Text, useFocus, useInput } from 'ink';
import { TelegramClient } from 'telegram';
import chalk from 'chalk';
import { componenetFocusIds } from '@/lib/utils/consts';
import { UserInfo } from '@/lib/types';

const messageActions = [
	{
		name: 'delete',
		description: 'are u sure you want to delte',
		deleteMessageShortCuts: {
			delete: 'y'
			//TODO: allow user to chose delete only for him or for everyone
		},
		action: async (client: TelegramClient, messageId: number, selectedUser: UserInfo) => {
			await deleteMessage(client, selectedUser, messageId);
		}
	}
] as const;

export const Modal: React.FC<{ onClose: () => void }> = ({ onClose }) => {
	const { isFocused } = useFocus({ autoFocus: true, id: componenetFocusIds.modal });
	const client = useTGCliStore((state) => state.client)!;
	const selectedUser = useTGCliStore((state) => state.selectedUser);
	const setMessageAction = useTGCliStore((state) => state.setMessageAction);
	const messageAction = useTGCliStore((state) => state.messageAction);

	const currentChatType = useTGCliStore((state) => state.currentChatType);

	const messageActionCurrentActiveKey = messageAction?.action;
	const { action, deleteMessageShortCuts, description } = messageActions.find(
		({ name }) => name === messageActionCurrentActiveKey
	)!;
	const { conversation, setConversation } = conversationStore((state) => state);

	useInput(async (_, key) => {
		if (key.escape) {
			onClose();
			return;
		}
		const messageId = messageAction?.id;
		if (!messageId || !selectedUser) {
			console.log(messageId, selectedUser);
			return;
		}
		if (currentChatType === 'PeerUser') {
			action(client, messageId, selectedUser as UserInfo);
		}
		const filterConversation = conversation.filter(({ id }) => id !== messageId);
		setConversation(filterConversation);
		setMessageAction(null);
		onClose();
	});

	const bgColor = chalk.bgBlue(''.repeat(80));

	return (
		<Box
			borderColor={isFocused ? 'blue' : ''}
			borderStyle="round"
			flexDirection="column"
			width={80}
			height={20}
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
				width={50}
				alignItems="center"
				justifyContent="center"
				position="absolute"
				marginTop={15}
				marginLeft={30}
				marginRight={30}
			>
				<Text color="blue" bold>
					{description}
				</Text>
				{Object.keys(deleteMessageShortCuts).map((key) => {
					return (
						<Box key={key} gap={2}>
							<Text color="red" bold>
								{key}
							</Text>
							<Text color="green">
								{deleteMessageShortCuts[key as keyof typeof deleteMessageShortCuts]}
							</Text>
						</Box>
					);
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
