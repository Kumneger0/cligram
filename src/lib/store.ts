import { FormattedMessage, TGCliStore } from '@/types';
import { create } from 'zustand';

export const conversationStore = create<{
	conversation: FormattedMessage[];
	setConversation: (conversation: FormattedMessage[]) => void;
}>((set) => ({
	conversation: [],
	setConversation: (conversation) => set({ conversation })
}));

export const useTGCliStore = create<TGCliStore>((set) => ({
	client: null,
	currentChatType: 'PeerUser',
	setCurrentChatType: (currentChatType) => set((state) => ({ ...state, currentChatType })),
	updateClient: (client) => set((state) => ({ ...state, client })),
	selectedUser: null,
	setSelectedUser: (selectedUser) => set((state) => ({ ...state, selectedUser })),
	messageAction: null,
	setMessageAction: (messageAction) => set((state) => ({ ...state, messageAction }))
}));
