import { css } from '@emotion/css';
import React from 'react';
import { Controller, useForm } from 'react-hook-form';

import { Correlation, GrafanaTheme2 } from '@grafana/data';
import { DataSourcePicker } from '@grafana/runtime';
import { Button, HorizontalGroup, PanelContainer, useStyles2 } from '@grafana/ui';
import { CloseButton } from 'app/core/components/CloseButton/CloseButton';

import { CorrelationDetailsFormPart } from './CorrelationDetailsFormPart';

const getStyles = (theme: GrafanaTheme2) => ({
  panelContainer: css`
    position: relative;
    padding: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(2)};
  `,
  buttonRow: css`
    display: flex;
    justify-content: flex-end;
  `,
});

interface FormDTO {
  source: string;
  target: string;
}

interface Props {
  onClose: () => void;
  onSubmit: (sourceUid: string, correlation: Correlation) => Promise<void>;
}

export const AddCorrelationForm = ({ onClose, onSubmit: externalSubmit }: Props) => {
  const styles = useStyles2(getStyles);
  const { handleSubmit, control } = useForm<FormDTO>();

  const onSubmit = handleSubmit(async (e) => {
    console.log(e);
    await externalSubmit(e.source, { target: e.target });
    onClose();
  });

  return (
    <PanelContainer className={styles.panelContainer}>
      <CloseButton onClick={onClose} />
      <form onSubmit={onSubmit}>
        <div>
          <HorizontalGroup>
            <Controller
              control={control}
              name="source"
              render={({ field: { onChange, value } }) => (
                <DataSourcePicker onChange={(v) => onChange(v.uid)} noDefault current={value} />
              )}
            />
            links to:
            <Controller
              control={control}
              name="target"
              render={({ field: { onChange, value } }) => (
                <DataSourcePicker onChange={(v) => onChange(v.uid)} noDefault current={value} />
              )}
            />
          </HorizontalGroup>
        </div>

        <CorrelationDetailsFormPart />

        <div className={styles.buttonRow}>
          <Button variant="primary" icon="plus" type="submit">
            Add
          </Button>
        </div>
      </form>
    </PanelContainer>
  );
};
