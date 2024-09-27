import blessed from 'blessed';
import { TelegramClient } from 'telegram';
import { chatUsers, getUserChats } from '../telegram/client.js';
import { screen, grid } from './initializeUI.js';
import { chatBox, updateChatBox } from './chatBox.js';
import { getConversationHistory } from '../telegram/messages.js';
import { addKeyBindingToFocusOnInputBox } from './inputBox.js';

export let sidebar: blessed.Widgets.ListElement;
export let selectedName: string;

export async function initializeSidebar(client: TelegramClient) {
    const chats = await getUserChats(client);
    sidebar = grid.set(0, 0, 12, 3, blessed.list, {
        label: 'Names',
        items: chats.map((user) => user?.firstName ?? null).filter(Boolean),
        keys: true,
        mouse: true,
        style: {
            selected: {
                bg: 'blue',
                fg: 'white'
            }
        }
    });

    sidebar.on('select', async (item: blessed.Widgets.ListElement) => {
        selectedName = item.getText();
        chatBox.setLabel(selectedName);
        const user = chatUsers.find(({ firstName }) => firstName === selectedName);
        if (user) {
            const conversation = await getConversationHistory(user);
            updateChatBox(conversation);
            addKeyBindingToFocusOnInputBox();
        }
        chatBox.focus();
        screen.render();
    });
}

export function rerenderSidebar(items: string[]) {
    screen.remove(sidebar);

    sidebar = grid.set(0, 0, 12, 3, blessed.list, {
        label: 'Names',
        items,
        keys: true,
        mouse: true,
        style: {
            selected: {
                bg: 'blue',
                fg: 'white'
            }
        }
    });
    screen.render();
}