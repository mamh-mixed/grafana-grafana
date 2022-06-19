import React from 'react';

import { LoadingState, PanelData } from '@grafana/data';

import { SceneDataNode } from './SceneDataNode';
import { SceneObjectBase } from './SceneObjectBase';
import { SceneComponentProps, SceneObject, SceneObjectList, SceneObjectState, SceneLayoutState } from './types';

interface RepeatOptions extends SceneObjectState {
  layout: SceneObject<SceneLayoutState>;
}

export class ScenePanelRepeater extends SceneObjectBase<RepeatOptions> {
  onMount() {
    super.onMount();

    this.subs.add(
      this.getData().subscribe({
        next: (data) => {
          if (data.data?.state === LoadingState.Done) {
            this.performRepeat(data.data);
          }
        },
      })
    );
  }

  performRepeat(data: PanelData) {
    // assume parent is a layout
    const firstChild = this.state.layout.state.children[0]!;
    const newChildren: SceneObjectList = [];

    for (const series of data.series) {
      const clone = firstChild.clone({
        key: `${newChildren.length}`,
        $data: new SceneDataNode({
          data: {
            ...data,
            series: [series],
          },
        }),
      });

      newChildren.push(clone);
    }

    this.state.layout.setState({ children: newChildren });
  }

  static Component = ({ model, isEditing }: SceneComponentProps<ScenePanelRepeater>) => {
    const { layout } = model.useMount().useState();
    return <layout.Component model={layout} isEditing={isEditing} />;
  };
}
