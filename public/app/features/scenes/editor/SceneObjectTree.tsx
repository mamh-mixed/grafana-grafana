import { css, cx } from '@emotion/css';
import React from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { Icon, useStyles2 } from '@grafana/ui';

import { SceneObject, SceneLayoutState, SceneObjectList, isSceneObject } from '../models/types';

export interface Props {
  node: SceneObject;
  selectedObject?: SceneObject;
}

export function SceneObjectTree({ node, selectedObject }: Props) {
  const styles = useStyles2(getStyles);
  const state = node.useState();
  let children: SceneObjectList = [];

  for (const propKey of Object.keys(state)) {
    const propValue = (state as any)[propKey];
    if (isSceneObject(propValue)) {
      children.push(propValue);
    }
  }

  let layoutChildren = (state as SceneLayoutState).children;
  if (layoutChildren) {
    for (const child of layoutChildren) {
      children.push(child);
    }
  }

  const name = node.constructor.name;
  const isSelected = selectedObject === node;
  const onSelectNode = () => node.getSceneEditor().selectObject(node);

  return (
    <div className={styles.node}>
      <div className={styles.header} onClick={onSelectNode}>
        <div className={styles.icon}>{children.length > 0 && <Icon name="angle-down" size="sm" />}</div>
        <div className={cx(styles.name, isSelected && styles.selected)}>{name}</div>
      </div>
      {children.length > 0 && (
        <div className={styles.children}>
          {children.map((child) => (
            <SceneObjectTree node={child} selectedObject={selectedObject} key={child.state.key} />
          ))}
        </div>
      )}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    node: css({
      display: 'flex',
      flexGrow: 0,
      cursor: 'pointer',
      flexDirection: 'column',
      padding: '2px 4px',
    }),
    header: css({
      display: 'flex',
      fontWeight: 500,
    }),
    name: css({}),
    selected: css({
      color: theme.colors.error.text,
    }),
    icon: css({
      width: theme.spacing(3),
      color: theme.colors.text.secondary,
    }),
    children: css({
      display: 'flex',
      flexDirection: 'column',
      paddingLeft: 8,
    }),
  };
};
