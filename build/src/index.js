#!/usr/bin/env node
var _a;
import blessed from "blessed";
import contrib from "blessed-contrib";
// Create a screen object.
const screen = blessed.screen({
    smartCSR: true,
    title: "CLI App with Sidebar",
});
// Create a layout grid.
const grid = new contrib.grid({ rows: 12, cols: 12, screen: screen });
// Create the sidebar list.
const sidebar = grid.set(0, 0, 12, 3, blessed.list, {
    label: "Names",
    items: ["Alice", "Bob", "Charlie", "David"],
    keys: true,
    mouse: true,
    style: {
        selected: {
            bg: "blue",
            fg: "white",
        },
    },
});
// Create a box to display selected names.
const displayBox = grid.set(0, 3, 12, 9, blessed.box, {
    label: "Selected Name",
    content: "Select a name from the sidebar",
    style: {
        fg: "green",
        border: {
            fg: "cyan",
        },
    },
});
(_a = sidebar === null || sidebar === void 0 ? void 0 : sidebar.on) === null || _a === void 0 ? void 0 : _a.call(sidebar, "select", (item) => {
    const selectedName = item.getText(); // Get the selected name
    displayBox.setContent(`Selected Name: ${selectedName}`); // Use the selected name
    screen.render();
});
screen.key(["escape", "q", "C-c"], function () {
    return process.exit(0);
});
sidebar.focus();
screen.render();
//# sourceMappingURL=index.js.map