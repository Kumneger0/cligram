import blessed from 'blessed';
import contrib from 'blessed-contrib';
import { TelegramClient } from 'telegram';
import { chatBox, initializeChatBox } from './chatBox.js';
import { initializeInputBox } from './inputBox.js';
import { initializeSidebar, namesList } from './sidebar.js';

export let screen: blessed.Widgets.Screen;
export let grid: contrib.grid;

export async function initializeUI(client: TelegramClient) {
    screen = blessed.screen({
        smartCSR: true,
        title: 'Terminal Telegram client'
    });

    grid = new contrib.grid({ rows: 12, cols: 12, screen: screen });

    await initializeSidebar(client);
    initializeChatBox();
    initializeInputBox(client);

    screen?.key(['escape', 'q', 'C-c'], () => process.exit(0));

    let focusOnSidebar = true;
    screen?.key(['tab'], () => {
        if (focusOnSidebar) {
            chatBox.focus();
        } else {
            namesList.focus();
        }
        focusOnSidebar = !focusOnSidebar;
        screen?.render();
    });

    screen?.render();
    namesList.focus();

}