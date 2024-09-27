
export interface ChatUser {
    firstName: string;
    isBot: boolean;
    peerId: bigInt.BigInteger;
    accessHash: bigInt.BigInteger;
}

export interface FormattedMessage {
    sender: string;
    content: string;
    isFromMe: boolean;
}

export const eventClassNames = ['UpdateUserStatus', 'UpdateShortMessage'] as const;