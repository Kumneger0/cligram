import { FormattedMessage } from '@/types.js';

const MAX_WIDTH = 50;

async function renderImage(imageBuffer: Buffer): Promise<string> {
	const terminalSize = (await import('term-size')).default;
	const terminalImage = (await import('terminal-image')).default;
	try {
		const { columns, rows } = terminalSize();
		const maxWidth = Math.floor(columns * 0.8); // Use 80% of terminal width
		const maxHeight = Math.floor(rows * 0.8);

		const string = await terminalImage.buffer(imageBuffer, {
			width: maxWidth,
			height: maxHeight,
			preserveAspectRatio: true
		});

		return string;
	} catch (error) {
		// const clipboard = (await import('clipboardy')).default;
		// clipboard.writeSync(
		// 	!!error &&
		// 		typeof error === 'object' &&
		// 		'message' in error &&
		// 		typeof error.message === 'string'
		// 		? error.message
		// 		: 'Unknown error'
		// );
		process.exit(1);
	}
}

export async function formatMessage(message: FormattedMessage): Promise<string> {
	const { sender, content, isFromMe, media } = message;
	const padding = isFromMe ? '' : '  ';
	const header = `${padding}${sender}:`;
	const wrappedContent = wrapText(content, MAX_WIDTH - padding.length);

	let formattedMessage = header + '\n';
	formattedMessage += wrappedContent.map((line) => `${padding}${line}`).join('\n');

	if (media) {
		const image = await renderImage(media);
		formattedMessage = image;
	}
	return formattedMessage;
}

export function wrapText(text: string, maxWidth: number): string[] {
	const words = text?.split(' ') ?? [];
	const lines: string[] = [];
	let currentLine = '';

	words.forEach((word) => {
		if (currentLine.length + word.length + 1 > maxWidth) {
			lines.push(currentLine);
			currentLine = word;
		} else {
			currentLine += (currentLine ? ' ' : '') + word;
		}
	});

	if (currentLine) {
		lines.push(currentLine);
	}

	return lines;
}
