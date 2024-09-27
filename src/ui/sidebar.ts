import blessed from 'blessed';
import { TelegramClient } from 'telegram';
import { chatUsers, getUserChats } from '../telegram/client.js';
import { screen, grid } from './initializeUI.js';
import { chatBox, updateChatBox } from './chatBox.js';
import { getConversationHistory } from '../telegram/messages.js';
import { addKeyBindingToFocusOnInputBox } from './inputBox.js';

export let sidebar: blessed.Widgets.BoxElement;
export let namesList: blessed.Widgets.ListElement;
export let keybindingsList: blessed.Widgets.ListElement;
export let selectedName: string;

export async function initializeSidebar(client: TelegramClient) {
    const chats = await getUserChats(client);

    sidebar = grid.set(0, 0, 12, 3, blessed.box, {
        label: 'Sidebar',
    });

    namesList = blessed.list({
        parent: sidebar,
        label: 'Names',
        top: 0,
        left: 0,
        width: '100%',
        height: '50%',
        items: chats.map((user) => user?.firstName ?? null).filter(Boolean),
        keys: true,
        vi: true,  
        mouse: true,
        style: {
            selected: {
                bg: 'blue',
                fg: 'white'
            }
        },
        scrollable: true,
        scrollbar: {
            ch: ' ',
            track: {
                bg: 'cyan'
            },
            style: {
                inverse: true
            }
        }
    });

    keybindingsList = blessed.list({
        parent: sidebar,
        label: 'Keybindings',
        top: '50%',
        left: 0,
        width: '100%',
        height: '50%',
        items: [
            'Navigation:',
            ' j / ↓ : Move down',
            ' k / ↑ : Move up',
            '',
            'Selection:',
            ' Enter : Select item',
            '',
            'Focus:',
            ' Tab : Switch focus',
            ' i : Focus input box',
            '',
            'Tip: Use arrow keys or',
            'j/k for navigation'
        ],
        style: {
            item: {
                fg: 'yellow'
            }
        }
    });

    namesList.on('select', async (item: blessed.Widgets.ListElement) => {
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

    namesList.focus();

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