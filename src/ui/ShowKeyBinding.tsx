import { Box, Text } from 'ink'
import React from 'react'

const keyBindings = {
    general: {
        'ctrl+k': 'Search',
        "Tab": 'Switch focus'
    },
    chatArea: {
        'r': 'Reply',
        'd': 'Delete message',
        'e': 'Edit message',
        'j': 'Go down',
        "k": "Go up",
    },
    sidebar: {
        'j': 'Go down',
        "k": "Go up",
        "c": 'Switch to channels',
        "u": 'Switch to users',
    }
}

type ShowKeyBindingProps = {
    type: keyof typeof keyBindings
}
function ShowKeyBinding({ type }: ShowKeyBindingProps) {
    const keyBindingToShow = type !== 'general' ? { ...keyBindings.general, ...keyBindings[type] } : keyBindings[type]
    return (
        <Box>
            <Text>
                {Object.entries(keyBindingToShow).map(([key, value]) => (
                    <Text color="blue" key={key}>{key} - {value}, {' '}</Text>
                ))}
            </Text>
        </Box>
    )
}

export default ShowKeyBinding