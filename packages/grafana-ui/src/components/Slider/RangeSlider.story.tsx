import { ComponentMeta, ComponentStory } from '@storybook/react';
import React from 'react';

import { RangeSlider } from '@grafana/ui';

const meta: ComponentMeta<typeof RangeSlider> = {
  title: 'Forms/Slider/Range',
  component: RangeSlider,
  parameters: {
    controls: {
      exclude: ['tooltipAlwaysVisible'],
    },
  },
  argTypes: {
    orientation: { control: { type: 'select', options: ['horizontal', 'vertical'] } },
    step: { control: { type: 'number', min: 1 } },
  },
  args: {
    min: 0,
    max: 100,
    orientation: 'horizontal',
    reverse: false,
    step: undefined,
  },
};

export const Basic: ComponentStory<typeof RangeSlider> = (args) => {
  return (
    <div style={{ width: '200px', height: '200px' }}>
      <RangeSlider value={[10, 62]} {...args} />
    </div>
  );
};

export const Vertical: ComponentStory<typeof RangeSlider> = (args) => {
  return (
    <div style={{ width: '200px', height: '200px' }}>
      <RangeSlider orientation="vertical" {...args} />
    </div>
  );
};

export default meta;
