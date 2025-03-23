import { create } from 'zustand';
import { FormattedMessage, ForwardMessageOptions, TGCliStore } from './types';

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
	searchMode: null,
	setSearchMode: (searchMode) => set((state) => ({ ...state, searchMode })),
	setCurrentChatType: (currentChatType) => set((state) => ({ ...state, currentChatType })),
	updateClient: (client) => set((state) => ({ ...state, client })),
	selectedUser: null,
	setSelectedUser: (selectedUser) => set((state) => ({ ...state, selectedUser })),
	messageAction: null,
	setMessageAction: (messageAction) => set((state) => ({ ...state, messageAction })),
	currentlyFocused: null,
	setCurrentlyFocused: (currentlyFocused) => set((state) => ({ ...state, currentlyFocused }))
}));


export const useForwardMessageStore = create<{
	forwardMessageOptions: ForwardMessageOptions | null;
	setForwardMessageOptions: (forwardMessageOptions: ForwardMessageOptions | null) => void;
}>((set) => ({
	forwardMessageOptions: null,
	setForwardMessageOptions: (forwardMessageOptions) => set((state) => ({ ...state, forwardMessageOptions }))
}));