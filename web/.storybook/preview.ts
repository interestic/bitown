import type { Preview } from "@storybook/react";
import "../src/styles/catalog.css";

const preview: Preview = {
  parameters: {
    layout: "fullscreen",
    controls: { expanded: true },
    options: {
      storySort: {
        order: [
          "Lot Patterns",
          ["Overview", "*"],
          "Placement Objects",
          ["Overview", "By Tag", "*"],
        ],
      },
    },
  },
};

export default preview;
