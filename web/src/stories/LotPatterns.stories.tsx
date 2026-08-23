import type { Meta, StoryObj } from "@storybook/react";
import { LotPatternView } from "../components/LotPatternView";
import { RoadDiamondGridView } from "../components/RoadDiamondGridView";
import { LOT_PATTERN_IDS } from "../catalog/lotPatterns";

const meta = {
  title: "Lot Patterns",
  component: LotPatternView,
  args: {
    scale: 3,
    showAnchor: false,
    variantIndex: 0,
    includeTreeObjects: true,
  },
  argTypes: {
    scale: { control: { type: "range", min: 1, max: 6, step: 0.5 } },
    variantIndex: {
      control: { type: "range", min: 0, max: 2, step: 1 },
      description:
        "702_mcRoad スタイル (0=細暗 / 1=土 / 2=アスファルト)。dir0→1..3 / dir1→4..6。Road Diamond のみ",
    },
    showAnchor: { control: "boolean" },
    includeTreeObjects: { control: "boolean" },
    patternId: {
      control: "select",
      options: [undefined, ...LOT_PATTERN_IDS],
    },
  },
} satisfies Meta<typeof LotPatternView>;

export default meta;
type Story = StoryObj<typeof meta>;

/** Ventura で見える空き地パターンを Game.hx size/type とクリップで対応づけた早見 */
export const Overview: Story = {
  args: {},
};

export const TreesPlot: Story = {
  args: { patternId: "trees-plot" },
};

export const GrassFill: Story = {
  args: { patternId: "grass-fill" },
};

export const YellowFill: Story = {
  args: { patternId: "yellow-fill" },
};

export const YellowFurrowNS: Story = {
  name: "Yellow furrow (vertical)",
  args: { patternId: "yellow-furrow-ns" },
};

export const YellowFurrowEW: Story = {
  name: "Yellow furrow (horizontal)",
  args: { patternId: "yellow-furrow-ew" },
};

export const Soil: Story = {
  args: { patternId: "soil" },
};

export const Pumpkin5: Story = {
  name: "Five pumpkins",
  args: { patternId: "pumpkin-5" },
};

export const QuadSplit: Story = {
  args: { patternId: "quad-split" },
};

/** 2×2 genSquare BIG ROADS / CROSS（マス単位スタンプ） */
export const RoadDiamond: Story = {
  name: "Road Diamond · genSquare",
  args: { patternId: "road-diamond", variantIndex: 2 },
};

export const RoadDiamondGrid: StoryObj<typeof RoadDiamondGridView> = {
  name: "Road Diamond · genSquare (standalone)",
  render: (args) => <RoadDiamondGridView {...args} />,
  args: { scale: 3, showAnchor: false, variantIndex: 2 },
  argTypes: {
    scale: { control: { type: "range", min: 1, max: 6, step: 0.5 } },
    variantIndex: {
      control: { type: "range", min: 0, max: 2, step: 1 },
      description: "702 スタイル (0=細暗 / 1=土 / 2=アスファルト)。dir0=1..3 / dir1=4..6",
    },
    showAnchor: { control: "boolean" },
  },
};

export const Dalle: Story = {
  args: { patternId: "dalle" },
};
