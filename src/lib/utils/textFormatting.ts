import { FormattedMessage } from '@/types.js';

const MAX_WIDTH = 50; // Approximate character width equivalent to 500px

export function formatMessage(message: FormattedMessage, index: number): string[] {
    const { sender, content, isFromMe } = message;
    const padding = isFromMe ? '' : '  ';
    const header = `${padding}${sender}:`;
    const wrappedContent = wrapText(content, MAX_WIDTH - padding.length);

    return [
        header,
        ...wrappedContent.map(line => `${padding}${line}`),
        '' // Empty line for spacing between messages
    ];
}

export function wrapText(text: string, maxWidth: number): string[] {
    const words = text.split(' ');
    const lines: string[] = [];
    let currentLine = '';

    words.forEach(word => {
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