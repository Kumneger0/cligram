import { formatMessage } from '@/lib/utils/textFormatting.js';
import blessed from 'blessed';
import * as types from '../types.js';
import * as initializeUI from './initializeUI.js';

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
		tags: true,
		itemHeight: 'auto',
		pageSize: 1000
	}) as blessed.Widgets.ListElement;

	chatBox.key(['up', 'down'], function (_ch, key) {
		const direction = key.name === 'up' ? -1 : 1;
		currentLine = Math.max(0, Math.min(currentLine + direction, chatBox.getLines().length - 1));
		updateChatBox();
		initializeUI.screen.render();
	});
}

export async function updateChatBox(newContent?: types.FormattedMessage[]) {
	if (newContent) {
		const formattedMessages = await Promise.all(
			newContent.flatMap((msg, _index) => formatMessage(msg))
		);

		chatBox.setContent(formattedMessages.join('\n'));

		(chatBox as blessed.Widgets.ListElement).setItems(formattedMessages);
		(chatBox as blessed.Widgets.ListElement).scrollTo(formattedMessages.length - 1);
		chatBox.setScrollPerc(100);
	}
	initializeUI.screen.render();
}




