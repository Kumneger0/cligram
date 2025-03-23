import { useTGCliStore } from '@/lib/store'
import { Box, Text } from 'ink'
import React from 'react'

const keyBindings = {
    general: {
        'ctrl+k': {
            mode: 'all',
            description: 'Search'
        },
        "Tab": {
            mode: 'all',
            description: 'Switch focus'
        }
    },
    chatArea: {
        'r': {
            mode: 'PeerUser',
            description: 'Reply'
        },
        'd': {
            mode: 'PeerUser',
            description: 'Delete message'
        },
        'e': {
            mode: 'PeerUser',
            description: 'Edit message'
        },
        'j': {
            mode: 'all',
            description: 'Go down'
        },
        "k": {
            mode: 'all',
            description: 'Go up'
        },
    },
    sidebar: {
        'j': {
            mode: 'all',
            description: 'Go down'
        },
        "k": {
            mode: 'all',
            description: 'Go up'
        },
        "c": {
            mode: 'PeerUser',
            description: 'Switch to channels'
        },
        "u": {
            mode: 'PeerChannel',
            description: 'Switch to users'
        }
    }
}

type ShowKeyBindingProps = {
    type: keyof typeof keyBindings
}
function ShowKeyBinding({ type }: ShowKeyBindingProps) {
    const currentChatType = useTGCliStore((state) => state.currentChatType)
    const keyBindingToShow = Object.entries(type !== 'general' ? { ...keyBindings.general, ...keyBindings[type] } : keyBindings[type]).filter(([_key, value]) => {
        if (value.mode === 'all') return true
        if (value.mode === currentChatType) return true
        return false
    })
    return (
        <Box>
            <Text>
                {keyBindingToShow.map(([key, value]) => (
                    <Text color="blue" key={key}>{key} - {value.description}, {' '}</Text>
                ))}
            </Text>
        </Box>
    )
}

export default ShowKeyBinding