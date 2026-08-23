import type { Meta, StoryObj } from "@storybook/react";
import { PlacementCatalogView } from "../components/PlacementCatalogView";
import { PLACEMENT_TAGS } from "../catalog/types";

const meta = {
  title: "Placement Objects",
  component: PlacementCatalogView,
  args: {
    scale: 2,
    showAnchor: true,
    variantIndex: 0,
    includeExclude: false,
  },
  argTypes: {
    scale: { control: { type: "range", min: 1, max: 4, step: 0.5 } },
    variantIndex: {
      control: { type: "range", min: 0, max: 3, step: 1 },
      description: "色バリアント (v00–v03)。各 clip-frame 内で切り替え",
    },
    showAnchor: { control: "boolean" },
    includeExclude: { control: "boolean" },
    tag: {
      control: "select",
      options: [undefined, ...PLACEMENT_TAGS, "exclude"],
    },
  },
} satisfies Meta<typeof PlacementCatalogView>;

export default meta;
type Story = StoryObj<typeof meta>;

/** 配置に使うタグをまとめた早見表 */
export const Overview: Story = {
  args: {},
};

export const Residential: Story = {
  args: { tag: "residential" },
};

export const Industrial: Story = {
  args: { tag: "industrial" },
};

export const Commercial: Story = {
  args: { tag: "commercial" },
};

export const Road: Story = {
  args: { tag: "road" },
};

export const Tree: Story = {
  args: { tag: "tree" },
};

export const Ground: Story = {
  args: { tag: "ground" },
};

export const Water: Story = {
  args: { tag: "water" },
};

export const Landmark: Story = {
  args: { tag: "landmark" },
};

export const Park: Story = {
  args: { tag: "park" },
};

/** マップ配置から除外された断片（件数が多い） */
export const Exclude: Story = {
  args: { tag: "exclude", scale: 1.5 },
};
