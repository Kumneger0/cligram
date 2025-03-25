import { create } from 'zustand';
import { FormattedMessage, ForwardMessageOptions, TGCliStore } from './types';

export const conversationStore = create<{
	conversation: FormattedMessage[];
	setConversation: (conversation: FormattedMessage[]) => void;
	updateConversations: (conversation: FormattedMessage[]) => void;
}>((set) => {return {
	conversation: [],
	setConversation: (conversation) => { return set(() => ({ conversation: conversation })) },
	updateConversations: (conversation) => { return set((state) => ({ conversation: [...state.conversation, ...conversation] })) }
}});

export const useTGCliStore = create<TGCliStore>((set) => {return {
	client: null,
	currentChatType: 'PeerUser',
	searchMode: null,
	setSearchMode: (searchMode) => {return set((state) => {return { ...state, searchMode }})},
	setCurrentChatType: (currentChatType) => {return set((state) => {return { ...state, currentChatType }})},
	updateClient: (client) => {return set((state) => {return { ...state, client }})},
	selectedUser: null,
	setSelectedUser: (selectedUser) => {return set((state) => {return { ...state, selectedUser }})},
	messageAction: null,
	setMessageAction: (messageAction) => {return set((state) => {return { ...state, messageAction }})},
	currentlyFocused: null,
	setCurrentlyFocused: (currentlyFocused) => {return set((state) => {return { ...state, currentlyFocused }})}
}});


export const useForwardMessageStore = create<{
	forwardMessageOptions: ForwardMessageOptions | null;
	setForwardMessageOptions: (forwardMessageOptions: ForwardMessageOptions | null) => void;
}>((set) => {return {
	forwardMessageOptions: null,
	setForwardMessageOptions: (forwardMessageOptions) => {return set((state) => {return { ...state, forwardMessageOptions }})}
}});