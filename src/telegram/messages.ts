import { Api, TelegramClient } from 'telegram';
import { MessagesSlice, User } from '../lib/types/index';
import { getTelegramClient } from '../lib/utils/auth';
import { ChatUser, FormattedMessage, eventClassNames } from '../types';
import { getUserChats, getUserInfo } from './client';

export async function getConversationHistory(
	{ accessHash, firstName, peerId: userId }: ChatUser,
	limit: number = 100
): Promise<FormattedMessage[]> {
	const client = await getTelegramClient();
	if (!client.connected) await client.connect();

	const result = (await client.invoke(
		new Api.messages.GetHistory({
			peer: new Api.InputPeerUser({
				userId: userId as unknown as bigInt.BigInteger,
				accessHash
			}),
			limit
		})
	)) as unknown as MessagesSlice;

	return await Promise.all(
		result.messages.reverse().map(async (message): Promise<FormattedMessage> => {
			// const media = message.media as MessageMedia

			return {
				id: message.id,
				sender: message.out ? 'you' : firstName,
				content: message.message,
				isFromMe: message.out,
				//TODO: implement media download later after updatin the layout
				media: null
				// media: media && media.className == 'MessageMediaPhoto' ? await downloadMedia({ media, size: 'large' }) : null,
			};
		})
	);
}
export const listenForUserMessages = async (
	client: TelegramClient,
	setUserChats: React.Dispatch<React.SetStateAction<ChatUser[]>>,
	selectedName: string
) => {
	if (!client.connected) await client.connect();
	console.log('Listening for messages');

	client.addEventHandler(async (event) => {
		const userId = event.userId;
		if (userId) {
			const user = (await getUserInfo(client, userId)) as unknown as User;

			const isNewMessage =
				(event.className as (typeof eventClassNames)[number]) === 'UpdateShortMessage';

			if (isNewMessage) {
				if (user.firstName !== selectedName) {
					const users = await getUserChats(client);
					const userChats = users
						.filter(({ isBot }) => !isBot)
						.map(({ firstName, ...rest }) => ({
							firstName: firstName === selectedName ? firstName + ' *' : firstName,
							...rest
						}));
					setUserChats(userChats);
				}
			}
		}
	});
};
