import { isArray } from 'lodash';
import React from 'react';

import { LoadingState } from '@grafana/data';
import { Select, MultiSelect } from '@grafana/ui';

import { SceneComponentProps } from '../../core/types';
import { MultiValueVariable } from '../variants/MultiValueVariable';

export function VariableValueSelect({ model }: SceneComponentProps<MultiValueVariable>) {
  const { value, key, state, isMulti, options } = model.useState();

  if (isMulti) {
    return (
      <MultiSelect
        id={key}
        placeholder="Select value"
        width="auto"
        value={isArray(value) ? value : [value]}
        allowCustomValue
        isLoading={state === LoadingState.Loading}
        options={options}
        onChange={model.onMultiValueChange}
      />
    );
  }

  return (
    <Select
      id={key}
      placeholder="Select value"
      width="auto"
      value={value}
      allowCustomValue
      isLoading={state === LoadingState.Loading}
      options={options}
      onChange={model.onSingleValueChange}
    />
  );
}
