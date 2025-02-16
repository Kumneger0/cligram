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
		const allMessgesToDisplay = newContent.filter(({ content }) => !!content)
			.map(({ content, isFromMe, sender }) => isFromMe ? content : `${sender} ${content}`);
		chatBox.setContent(allMessgesToDisplay.join('\n'));
		(chatBox as blessed.Widgets.ListElement).scrollTo(newContent.length - 1);
		chatBox.setScrollPerc(100);
	}
	initializeUI.screen.render();
}




