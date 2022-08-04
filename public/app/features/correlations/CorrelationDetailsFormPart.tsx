import React from 'react';

import { Field, Input, TextArea } from '@grafana/ui';

export const CorrelationDetailsFormPart = () => {
  return (
    <>
      <Field label="Labels">
        <Input id="lol1" />
      </Field>

      <Field label="Description">
        <TextArea id="lol2" />
      </Field>
    </>
  );
};
