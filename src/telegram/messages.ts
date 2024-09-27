import { TelegramClient, Api } from 'telegram';
import { MessagesSlice, User } from '../lib/types/index.js';
import { ChatUser, FormattedMessage, eventClassNames } from '../types.js';
import { getUserChats, getUserInfo } from './client.js';
import { rerenderSidebar, selectedName } from '../ui/sidebar.js';
import { getTelegramClient } from '../lib/utils/auth.js';

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

    return result.messages.reverse().map(
        (message): FormattedMessage => ({
            sender: message.out ? 'you' : firstName,
            content: message.message,
            isFromMe: message.out
        })
    );
}

export const listenForUserMessages = async (client: TelegramClient) => {
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
                        .map(({ firstName }) => firstName)
                        .map((name) => (name === selectedName ? name + ' *' : name));

                    rerenderSidebar(userChats);
                }
            }
        }
    });
};