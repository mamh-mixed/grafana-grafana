import { Story, Meta } from '@storybook/react';
import React, { useState, ChangeEvent } from 'react';

import { withCenteredStory } from '../../utils/storybook/withCenteredStory';

import { SecretTextArea, Props } from './SecretTextArea';

export default {
  title: 'Forms/SecretTextArea',
  component: SecretTextArea,
  decorators: [withCenteredStory],
  parameters: {
    controls: {
      exclude: [
        'prefix',
        'suffix',
        'addonBefore',
        'addonAfter',
        'type',
        'disabled',
        'invalid',
        'loading',
        'before',
        'after',
      ],
    },
  },
  args: {
    rows: 3,
    cols: 30,
    placeholder: 'Enter your secret...',
  },
  argTypes: {
    rows: { control: { type: 'range', min: 1, max: 50, step: 1 } },
    cols: { control: { type: 'range', min: 1, max: 200, step: 10 } },
  },
} as Meta;

const Template: Story<Props> = (args) => {
  const [secret, setSecret] = useState('');

  return (
    <SecretTextArea
      rows={args.rows}
      cols={args.cols}
      value={secret}
      isConfigured={args.isConfigured}
      placeholder={args.placeholder}
      onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setSecret(event.target.value.trim())}
      onReset={() => setSecret('')}
    />
  );
};

export const basic = Template.bind({});

basic.args = {
  isConfigured: false,
};

export const secretIsConfigured = Template.bind({});

secretIsConfigured.args = {
  isConfigured: true,
};
