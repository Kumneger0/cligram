import { useTGCliStore } from '@/lib/store';
import { ChannelInfo, UserInfo } from '@/lib/types';
import { ICONS } from '@/lib/utils';
import { componenetFocusIds } from '@/lib/utils/consts';
import { searchUsers } from '@/telegram/client';
import chalk from 'chalk';
import debounce from 'debounce';
import { Box, Text, useFocus, useInput } from 'ink';
import TextInput from 'ink-text-input';
import React, { useState } from 'react';

type SearchResult = { type: 'user'; data: UserInfo } | { type: 'channel'; data: ChannelInfo };

export const SearchModal: React.FC<{ height: number; width: number }> = ({ height, width }) => {
	const { isFocused } = useFocus({ autoFocus: true, id: componenetFocusIds.searchModal });
	const [query, setQuery] = useState('');
	const setSearchMode = useTGCliStore((state) => {
		return state.setSearchMode;
	});
	const searchMode = useTGCliStore((state) => {
		return state.searchMode;
	});
	const client = useTGCliStore((state) => {
		return state.client;
	});
	const currentChatType = useTGCliStore((state) => {
		return state.currentChatType;
	});
	const setCurrentChatType = useTGCliStore((state) => {
		return state.setCurrentChatType;
	});

	const setSelectedUser = useTGCliStore((state) => {
		return state.setSelectedUser;
	});

	const [combinedResults, setCombinedResults] = useState<SearchResult[]>([]);
	const [activeIndex, setActiveIndex] = useState<number>(-1);

	const debouncedSearch = debounce(async (query: string) => {
		if (query.length < 3) {
			return;
		}
		if (searchMode === 'CHANNELS_OR_ USERS') {
			const results = await searchUsers(client!, query);
			const combined: SearchResult[] = [
				...results.users.map((user) => {
					return { type: 'user' as const, data: user };
				}),
				...results.channels.map((channel) => {
					return { type: 'channel' as const, data: channel };
				})
			];
			setCombinedResults(combined);
			setActiveIndex(combined.length > 0 ? 0 : -1);
		}
		if (searchMode === 'CONVERSATION') {
			// TODO: Implement conversation search
			// const peerInfo =
			// 	currentChatType == 'PeerUser'
			// 		? {
			// 				accessHash: selectedUser!.accessHash,
			// 				peerId: (selectedUser! as UserInfo).peerId,
			// 				userFirtNameOrChannelTitle: (selectedUser! as UserInfo).firstName
			// 			}
			// 		: {
			// 				accessHash: selectedUser!.accessHash,
			// 				peerId: (selectedUser! as ChannelInfo).channelId,
			// 				userFirtNameOrChannelTitle: (selectedUser! as ChannelInfo).title
			// 			};
			// const messages = await getAllMessages(
			// 	{ client: client!, peerInfo, offsetId: 0, chatAreaWidth: 0 },
			// 	currentChatType,
			// 	{ search: query }
			// );
		}
	}, 1000);

	useInput((input, key) => {
		if (key.escape) {
			setSearchMode(null);
			return;
		}

		if (combinedResults.length === 0) {
			return;
		}

		if (key.return) {
			const result = combinedResults[activeIndex];
			if (result?.type === 'user') {
				if (currentChatType !== 'PeerUser') {
					setCurrentChatType('PeerUser');
				}
				setSelectedUser(result.data);
			}
			if (result?.type === 'channel') {
				if (currentChatType !== 'PeerChannel') {
					setCurrentChatType('PeerChannel');
				}
				setSelectedUser(result.data);
			}
			setSearchMode(null);
		}
		if (key.upArrow || input === 'k') {
			setActiveIndex((prev) => {
				return Math.max(0, prev - 1);
			});
		}
		if (key.downArrow || input === 'j') {
			setActiveIndex((prev) => {
				return Math.min(combinedResults.length - 1, prev + 1);
			});
		}
	});

	const bgColor = chalk.bgBlue(''.repeat(80));

	const modalBackadropWidth = width * (80 / 100);
	const modalBackadropHight = height * (80 / 100);

	return (
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
				width={modalBackadropWidth * 0.8}
				height={modalBackadropHight * 0.8}
				alignItems="center"
				justifyContent="center"
				position="absolute"
				marginTop={5}
				marginLeft={30}
				marginRight={30}
			>
				<Text color="blue" bold>
					{searchMode === 'CHANNELS_OR_ USERS' ? 'Search Channels or Users' : 'Search Conversation'}
				</Text>
				<Box marginTop={1} width="100%" alignItems="center">
					<Text color="white">{ICONS.SEARCH}</Text>
					<TextInput
						value={query}
						onChange={(value) => {
							setQuery(value);
							if (searchMode === 'CHANNELS_OR_ USERS') {
								debouncedSearch(value);
							}
						}}
						onSubmit={(value) => {
							if (searchMode === 'CONVERSATION') {
								debouncedSearch(value);
							}
						}}
						placeholder="Type to search..."
						focus={isFocused}
					/>
				</Box>
				<Box flexDirection="column">
					{searchMode === 'CHANNELS_OR_ USERS' &&
						combinedResults.map((result, index) => {
							return (
								<Text
									key={
										result.type === 'user' ? result.data.peerId.toString() : result.data.channelId
									}
									color={activeIndex === index ? 'green' : 'white'}
								>
									{activeIndex === index ? '> ' : '  '}
									{result.type === 'user'
										? `${ICONS.USER} ${result.data.firstName}`
										: `${ICONS.CHANNEL} ${result.data.title}`}
								</Text>
							);
						})}
				</Box>
				<Box marginTop={2}>
					<Text backgroundColor={'blue'} color={'white'}>
						(Press ESC to close, Enter to search)
					</Text>
				</Box>
				<Box marginTop={1} flexDirection="column" alignItems="center">
					<Text color="gray">Search Navigation:</Text>
					<Box gap={2}>
						<Text color="red" bold>
							j/k
						</Text>
						<Text color="green">navigate results</Text>
					</Box>
					<Box gap={2}>
						<Text color="red" bold>
							enter
						</Text>
						<Text color="green">select</Text>
					</Box>
				</Box>
			</Box>
		</Box>
	);
};
