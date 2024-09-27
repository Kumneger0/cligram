import { formatMessage } from '@/lib/utils/textFormatting.js';
import blessed from 'blessed';
import * as initializeUI from './initializeUI.js';
import * as types from '../types.js';


export let chatBox: blessed.Widgets.ListElement;
let currentLine = 0;

export function initializeChatBox() {
    chatBox = initializeUI.grid.set(0, 3, 10, 9, blessed.list, {
        label: 'Chat',
        scrollable: true,
        alwaysScroll: true,
        scrollbar: {
            ch: ' ',
            inverse: true
        },
        keys: true,
        vi: true,
        mouse: true,
        border: {
            type: 'line'
        },
        style: {
            fg: 'white',
            border: {
                fg: 'cyan'
            },
            selected: {
                bg: 'blue',
                fg: 'white'
            }
        },
        tags: true
    }) as blessed.Widgets.ListElement;

    chatBox.key(['up', 'down'], function (_ch, key) {
        const direction = key.name === 'up' ? -1 : 1;
        currentLine = Math.max(0, Math.min(currentLine + direction, chatBox.getLines().length - 1));
        updateChatBox();
        initializeUI.screen.render();
    });
}

export function updateChatBox(newContent?: types.FormattedMessage[]) {
    if (newContent) {
        const formattedMessages = newContent.flatMap((msg, index) => formatMessage(msg, index));
        (chatBox as blessed.Widgets.ListElement).setItems(formattedMessages);
        (chatBox as blessed.Widgets.ListElement).scrollTo(formattedMessages.length - 1);
    }
    initializeUI.screen.render();
}