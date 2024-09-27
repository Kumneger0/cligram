import { Api } from "telegram";

export function formatBytes(bytes: number) {
    const KB = 1024;
    const MB = KB * 1024;
    const GB = MB * 1024;

    if (bytes < KB) return `${bytes} Bytes`;
    if (bytes < MB) return `${(bytes / KB).toFixed(2)} KB`;
    if (bytes < GB) return `${(bytes / MB).toFixed(2)} MB`;

    return `${(bytes / GB).toFixed(2)} GB`;
}



export const getChannelEntity = (channelId: string, accessHash: string) => {
    return new Api.InputChannel({
        //@ts-ignore
        channelId: channelId,
        //@ts-ignore
        accessHash: accessHash
    });
};