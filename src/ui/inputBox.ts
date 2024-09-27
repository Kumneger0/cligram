import * as blessed from 'blessed';
import { TelegramClient, Api } from 'telegram';
import { grid, screen } from './initializeUI.js';
import { chatBox } from './chatBox.js';
import { selectedName, sidebar } from './sidebar.js';
import { chatUsers } from '../telegram/client.js';

export let inputBox: blessed.Widgets.InputElement;

export function initializeInputBox(client: TelegramClient) {
    inputBox = grid.set(
        10,
        3,
        2,
        9,
        blessed.textbox,
        {
            inputOnFocus: true,
            border: {
                type: "line",
            },
            style: {
                fg: "white",
                border: {
                    fg: "cyan",
                },
            }
        }
    );

    inputBox.key(['enter'], async () => {
        const value = inputBox.content;

        if (!client.connected) await client.connect();

        const userToSend = chatUsers.find(({ firstName }) => firstName == selectedName);
        if (!value || !userToSend) {
            chatBox.focus();
            return;
        }

        client.sendMessage(
            new Api.InputPeerUser({
                userId: userToSend?.peerId,
                accessHash: userToSend?.accessHash
            }),
            {
                message: value
            }
        );

        screen.render();
    });

    inputBox.key(['C-h'], () => {
        sidebar.focus();
    });
}

export function addKeyBindingToFocusOnInputBox() {
    if (chatBox && inputBox) {
        chatBox.key(['i'], () => {
            if (selectedName) {
                inputBox.focus();
                screen.render();
            }
        });
    }
}